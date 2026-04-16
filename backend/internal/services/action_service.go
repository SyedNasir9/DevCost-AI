package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"devcost-ai/pkg/logger"
)

// ActionService executes AWS resource actions
type ActionService struct {
	logger    *logger.Logger
	ec2Client *ec2.Client
	rdsClient *rds.Client
	config    *ActionConfig
}

// ActionConfig holds configuration for action execution
type ActionConfig struct {
	// Retry configuration
	MaxRetries int
	RetryDelay time.Duration

	// Timeouts
	EC2StopTimeout   time.Duration
	EBSDeleteTimeout time.Duration
	RDSResizeTimeout time.Duration

	// Safety settings
	DryRun              bool // If true, only simulate actions
	RequireConfirmation bool // If true, require explicit confirmation
}

// DefaultActionConfig returns default configuration
func DefaultActionConfig() *ActionConfig {
	return &ActionConfig{
		MaxRetries:          3,
		RetryDelay:          5 * time.Second,
		EC2StopTimeout:      2 * time.Minute,
		EBSDeleteTimeout:    1 * time.Minute,
		RDSResizeTimeout:    10 * time.Minute,
		DryRun:              false,
		RequireConfirmation: false,
	}
}

// ActionResult represents the result of an action execution
type ActionResult struct {
	ActionID     uuid.UUID              `json:"action_id"`
	ActionType   string                 `json:"action_type"` // stop_ec2, delete_ebs, resize_rds
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"` // EC2, EBS, RDS
	Status       string                 `json:"status"`        // pending, in_progress, success, failed
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Message      string                 `json:"message"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	DryRun       bool                   `json:"dry_run"`
}

// EC2StopInput holds input for stopping an EC2 instance
type EC2StopInput struct {
	InstanceID string
	Force      bool // Force stop (equivalent to unplugging)
	Hibernate  bool // Hibernate instead of stop (if enabled)
	DryRun     bool // Validate only, don't execute
}

// EBSDeleteInput holds input for deleting an EBS volume
type EBSDeleteInput struct {
	VolumeID       string
	CreateSnapshot bool // Create snapshot before deletion
	DryRun         bool // Validate only, don't execute
}

// RDSResizeInput holds input for resizing an RDS instance
type RDSResizeInput struct {
	InstanceID       string
	NewInstanceClass string // e.g., "db.t3.micro"
	ApplyImmediately bool   // Apply during next maintenance window if false
	DryRun           bool   // Validate only, don't execute
}

// NewActionService creates a new action service
func NewActionService(logger *logger.Logger, ec2Client *ec2.Client, rdsClient *rds.Client, config *ActionConfig) *ActionService {
	if config == nil {
		config = DefaultActionConfig()
	}

	return &ActionService{
		logger:    logger,
		ec2Client: ec2Client,
		rdsClient: rdsClient,
		config:    config,
	}
}

// StopEC2 stops an EC2 instance
func (s *ActionService) StopEC2(ctx context.Context, input EC2StopInput) (*ActionResult, error) {
	actionID := uuid.New()
	startTime := time.Now()

	result := &ActionResult{
		ActionID:     actionID,
		ActionType:   "stop_ec2",
		ResourceID:   input.InstanceID,
		ResourceType: "EC2",
		Status:       "pending",
		StartTime:    startTime,
		DryRun:       input.DryRun || s.config.DryRun,
	}

	s.logger.Info("Executing StopEC2 action",
		zap.String("action_id", actionID.String()),
		zap.String("instance_id", input.InstanceID),
		zap.Bool("force", input.Force),
		zap.Bool("hibernate", input.Hibernate),
		zap.Bool("dry_run", result.DryRun),
	)

	// Check current instance state first
	describeInput := &ec2.DescribeInstancesInput{
		InstanceIds: []string{input.InstanceID},
	}

	describeOutput, err := s.ec2Client.DescribeInstances(ctx, describeInput)
	if err != nil {
		s.logger.Error("Failed to describe EC2 instance",
			zap.String("action_id", actionID.String()),
			zap.String("instance_id", input.InstanceID),
			zap.Error(err),
		)
		return s.failResult(result, startTime, "Failed to describe instance", err), err
	}

	if len(describeOutput.Reservations) == 0 || len(describeOutput.Reservations[0].Instances) == 0 {
		err := fmt.Errorf("instance not found: %s", input.InstanceID)
		s.logger.Error("EC2 instance not found",
			zap.String("action_id", actionID.String()),
			zap.String("instance_id", input.InstanceID),
		)
		return s.failResult(result, startTime, "Instance not found", err), err
	}

	instance := describeOutput.Reservations[0].Instances[0]
	currentState := instance.State.Name

	s.logger.Info("EC2 instance state check",
		zap.String("action_id", actionID.String()),
		zap.String("instance_id", input.InstanceID),
		zap.String("current_state", string(currentState)),
	)

	// Check if already stopped or stopping
	if currentState == types.InstanceStateNameStopped {
		msg := "Instance is already stopped"
		s.logger.Info(msg,
			zap.String("action_id", actionID.String()),
			zap.String("instance_id", input.InstanceID),
		)
		return s.successResult(result, startTime, msg, map[string]interface{}{
			"previous_state": "stopped",
		}), nil
	}

	if currentState == types.InstanceStateNameStopping {
		msg := "Instance is already in stopping state"
		s.logger.Info(msg,
			zap.String("action_id", actionID.String()),
			zap.String("instance_id", input.InstanceID),
		)
		return s.successResult(result, startTime, msg, map[string]interface{}{
			"previous_state": "stopping",
		}), nil
	}

	// Check if instance is in a state that can be stopped
	if currentState != types.InstanceStateNameRunning {
		err := fmt.Errorf("instance must be running to stop, current state: %s", currentState)
		s.logger.Error("Cannot stop EC2 instance - invalid state",
			zap.String("action_id", actionID.String()),
			zap.String("instance_id", input.InstanceID),
			zap.String("current_state", string(currentState)),
		)
		return s.failResult(result, startTime, fmt.Sprintf("Invalid instance state: %s", currentState), err), err
	}

	result.Status = "in_progress"

	// If dry run, just return success without executing
	if result.DryRun {
		msg := fmt.Sprintf("Dry run: Would stop instance %s", input.InstanceID)
		s.logger.Info(msg,
			zap.String("action_id", actionID.String()),
			zap.String("instance_id", input.InstanceID),
		)
		return s.successResult(result, startTime, msg, map[string]interface{}{
			"dry_run":        true,
			"previous_state": string(currentState),
		}), nil
	}

	// Execute stop
	stopInput := &ec2.StopInstancesInput{
		InstanceIds: []string{input.InstanceID},
		Force:       aws.Bool(input.Force),
		Hibernate:   aws.Bool(input.Hibernate),
	}

	stopOutput, err := s.ec2Client.StopInstances(ctx, stopInput)
	if err != nil {
		s.logger.Error("Failed to stop EC2 instance",
			zap.String("action_id", actionID.String()),
			zap.String("instance_id", input.InstanceID),
			zap.Error(err),
		)
		return s.failResult(result, startTime, "Failed to stop instance", err), err
	}

	// Verify the state change
	if len(stopOutput.StoppingInstances) > 0 {
		newState := stopOutput.StoppingInstances[0].CurrentState.Name
		previousState := stopOutput.StoppingInstances[0].PreviousState.Name

		s.logger.Info("EC2 instance stop initiated",
			zap.String("action_id", actionID.String()),
			zap.String("instance_id", input.InstanceID),
			zap.String("previous_state", string(previousState)),
			zap.String("new_state", string(newState)),
		)

		return s.successResult(result, startTime, "Instance stop initiated", map[string]interface{}{
			"previous_state": string(previousState),
			"current_state":  string(newState),
		}), nil
	}

	return s.successResult(result, startTime, "Stop command sent", nil), nil
}

// DeleteEBS deletes an EBS volume
func (s *ActionService) DeleteEBS(ctx context.Context, input EBSDeleteInput) (*ActionResult, error) {
	actionID := uuid.New()
	startTime := time.Now()

	result := &ActionResult{
		ActionID:     actionID,
		ActionType:   "delete_ebs",
		ResourceID:   input.VolumeID,
		ResourceType: "EBS",
		Status:       "pending",
		StartTime:    startTime,
		DryRun:       input.DryRun || s.config.DryRun,
	}

	s.logger.Info("Executing DeleteEBS action",
		zap.String("action_id", actionID.String()),
		zap.String("volume_id", input.VolumeID),
		zap.Bool("create_snapshot", input.CreateSnapshot),
		zap.Bool("dry_run", result.DryRun),
	)

	// Describe volume to check state
	describeInput := &ec2.DescribeVolumesInput{
		VolumeIds: []string{input.VolumeID},
	}

	describeOutput, err := s.ec2Client.DescribeVolumes(ctx, describeInput)
	if err != nil {
		s.logger.Error("Failed to describe EBS volume",
			zap.String("action_id", actionID.String()),
			zap.String("volume_id", input.VolumeID),
			zap.Error(err),
		)
		return s.failResult(result, startTime, "Failed to describe volume", err), err
	}

	if len(describeOutput.Volumes) == 0 {
		err := fmt.Errorf("volume not found: %s", input.VolumeID)
		s.logger.Error("EBS volume not found",
			zap.String("action_id", actionID.String()),
			zap.String("volume_id", input.VolumeID),
		)
		return s.failResult(result, startTime, "Volume not found", err), err
	}

	volume := describeOutput.Volumes[0]
	currentState := volume.State

	s.logger.Info("EBS volume state check",
		zap.String("action_id", actionID.String()),
		zap.String("volume_id", input.VolumeID),
		zap.String("current_state", string(currentState)),
		zap.Int("size_gb", int(*volume.Size)),
	)

	// Check if volume is attached
	if currentState == types.VolumeStateInUse {
		err := fmt.Errorf("volume %s is currently attached to an instance", input.VolumeID)
		s.logger.Error("Cannot delete EBS volume - volume is in use",
			zap.String("action_id", actionID.String()),
			zap.String("volume_id", input.VolumeID),
		)
		return s.failResult(result, startTime, "Volume is attached to an instance", err), err
	}

	// Check if already deleted
	if currentState == types.VolumeStateDeleting || currentState == types.VolumeStateDeleted {
		msg := fmt.Sprintf("Volume is already %s", string(currentState))
		s.logger.Info(msg,
			zap.String("action_id", actionID.String()),
			zap.String("volume_id", input.VolumeID),
		)
		return s.successResult(result, startTime, msg, map[string]interface{}{
			"previous_state": string(currentState),
		}), nil
	}

	// Only available volumes can be deleted
	if currentState != types.VolumeStateAvailable {
		err := fmt.Errorf("volume must be in 'available' state to delete, current state: %s", currentState)
		s.logger.Error("Cannot delete EBS volume - invalid state",
			zap.String("action_id", actionID.String()),
			zap.String("volume_id", input.VolumeID),
			zap.String("current_state", string(currentState)),
		)
		return s.failResult(result, startTime, fmt.Sprintf("Invalid volume state: %s", currentState), err), err
	}

	// Create snapshot if requested
	var snapshotID string
	if input.CreateSnapshot {
		s.logger.Info("Creating snapshot before volume deletion",
			zap.String("action_id", actionID.String()),
			zap.String("volume_id", input.VolumeID),
		)

		snapshotInput := &ec2.CreateSnapshotInput{
			VolumeId:    aws.String(input.VolumeID),
			Description: aws.String(fmt.Sprintf("Pre-deletion snapshot for %s", input.VolumeID)),
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeSnapshot,
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("backup-%s", input.VolumeID))},
						{Key: aws.String("SourceVolume"), Value: aws.String(input.VolumeID)},
						{Key: aws.String("CreatedBy"), Value: aws.String("devcost-ai")},
					},
				},
			},
		}

		snapshotOutput, err := s.ec2Client.CreateSnapshot(ctx, snapshotInput)
		if err != nil {
			s.logger.Error("Failed to create snapshot",
				zap.String("action_id", actionID.String()),
				zap.String("volume_id", input.VolumeID),
				zap.Error(err),
			)
			return s.failResult(result, startTime, "Failed to create snapshot", err), err
		}

		snapshotID = *snapshotOutput.SnapshotId
		s.logger.Info("Snapshot created",
			zap.String("action_id", actionID.String()),
			zap.String("volume_id", input.VolumeID),
			zap.String("snapshot_id", snapshotID),
			zap.String("state", string(snapshotOutput.State)),
		)

		result.Metadata = map[string]interface{}{
			"snapshot_id": snapshotID,
		}
	}

	result.Status = "in_progress"

	// If dry run, just return success without executing
	if result.DryRun {
		msg := fmt.Sprintf("Dry run: Would delete volume %s", input.VolumeID)
		s.logger.Info(msg,
			zap.String("action_id", actionID.String()),
			zap.String("volume_id", input.VolumeID),
		)
		return s.successResult(result, startTime, msg, map[string]interface{}{
			"dry_run":        true,
			"previous_state": string(currentState),
			"snapshot_id":    snapshotID,
		}), nil
	}

	// Execute delete
	deleteInput := &ec2.DeleteVolumeInput{
		VolumeId: aws.String(input.VolumeID),
	}

	_, err = s.ec2Client.DeleteVolume(ctx, deleteInput)
	if err != nil {
		s.logger.Error("Failed to delete EBS volume",
			zap.String("action_id", actionID.String()),
			zap.String("volume_id", input.VolumeID),
			zap.Error(err),
		)
		return s.failResult(result, startTime, "Failed to delete volume", err), err
	}

	s.logger.Info("EBS volume deletion initiated",
		zap.String("action_id", actionID.String()),
		zap.String("volume_id", input.VolumeID),
		zap.String("previous_state", string(currentState)),
	)

	return s.successResult(result, startTime, "Volume deletion initiated", map[string]interface{}{
		"previous_state": string(currentState),
		"snapshot_id":    snapshotID,
	}), nil
}

// ResizeRDS resizes an RDS instance
func (s *ActionService) ResizeRDS(ctx context.Context, input RDSResizeInput) (*ActionResult, error) {
	actionID := uuid.New()
	startTime := time.Now()

	result := &ActionResult{
		ActionID:     actionID,
		ActionType:   "resize_rds",
		ResourceID:   input.InstanceID,
		ResourceType: "RDS",
		Status:       "pending",
		StartTime:    startTime,
		DryRun:       input.DryRun || s.config.DryRun,
	}

	s.logger.Info("Executing ResizeRDS action",
		zap.String("action_id", actionID.String()),
		zap.String("instance_id", input.InstanceID),
		zap.String("new_instance_class", input.NewInstanceClass),
		zap.Bool("apply_immediately", input.ApplyImmediately),
		zap.Bool("dry_run", result.DryRun),
	)

	// Describe instance to check current state
	describeInput := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(input.InstanceID),
	}

	describeOutput, err := s.rdsClient.DescribeDBInstances(ctx, describeInput)
	if err != nil {
		s.logger.Error("Failed to describe RDS instance",
			zap.String("action_id", actionID.String()),
			zap.String("instance_id", input.InstanceID),
			zap.Error(err),
		)
		return s.failResult(result, startTime, "Failed to describe instance", err), err
	}

	if len(describeOutput.DBInstances) == 0 {
		err := fmt.Errorf("instance not found: %s", input.InstanceID)
		s.logger.Error("RDS instance not found",
			zap.String("action_id", actionID.String()),
			zap.String("instance_id", input.InstanceID),
		)
		return s.failResult(result, startTime, "Instance not found", err), err
	}

	instance := describeOutput.DBInstances[0]
	currentClass := *instance.DBInstanceClass
	currentStatus := *instance.DBInstanceStatus

	s.logger.Info("RDS instance check",
		zap.String("action_id", actionID.String()),
		zap.String("instance_id", input.InstanceID),
		zap.String("current_class", currentClass),
		zap.String("current_status", currentStatus),
		zap.String("target_class", input.NewInstanceClass),
	)

	// Check if already at target size
	if currentClass == input.NewInstanceClass {
		msg := fmt.Sprintf("Instance is already %s", input.NewInstanceClass)
		s.logger.Info(msg,
			zap.String("action_id", actionID.String()),
			zap.String("instance_id", input.InstanceID),
		)
		return s.successResult(result, startTime, msg, map[string]interface{}{
			"previous_class": currentClass,
			"current_class":  currentClass,
		}), nil
	}

	// Check if instance is in a modifiable state
	validStates := map[string]bool{
		"available":            true,
		"storage-optimization": true,
	}

	if !validStates[currentStatus] {
		err := fmt.Errorf("instance cannot be modified in current state: %s", currentStatus)
		s.logger.Error("Cannot resize RDS instance - invalid state",
			zap.String("action_id", actionID.String()),
			zap.String("instance_id", input.InstanceID),
			zap.String("current_status", currentStatus),
		)
		return s.failResult(result, startTime, fmt.Sprintf("Invalid instance status: %s", currentStatus), err), err
	}

	result.Status = "in_progress"

	// If dry run, just return success without executing
	if result.DryRun {
		msg := fmt.Sprintf("Dry run: Would resize instance %s from %s to %s",
			input.InstanceID, currentClass, input.NewInstanceClass)
		s.logger.Info(msg,
			zap.String("action_id", actionID.String()),
			zap.String("instance_id", input.InstanceID),
		)
		return s.successResult(result, startTime, msg, map[string]interface{}{
			"dry_run":        true,
			"previous_class": currentClass,
			"target_class":   input.NewInstanceClass,
		}), nil
	}

	// Execute resize
	modifyInput := &rds.ModifyDBInstanceInput{
		DBInstanceIdentifier: aws.String(input.InstanceID),
		DBInstanceClass:      aws.String(input.NewInstanceClass),
		ApplyImmediately:     aws.Bool(input.ApplyImmediately),
	}

	modifyOutput, err := s.rdsClient.ModifyDBInstance(ctx, modifyInput)
	if err != nil {
		s.logger.Error("Failed to modify RDS instance",
			zap.String("action_id", actionID.String()),
			zap.String("instance_id", input.InstanceID),
			zap.Error(err),
		)
		return s.failResult(result, startTime, "Failed to resize instance", err), err
	}

	newClass := *modifyOutput.DBInstance.DBInstanceClass
	newStatus := *modifyOutput.DBInstance.DBInstanceStatus

	s.logger.Info("RDS instance resize initiated",
		zap.String("action_id", actionID.String()),
		zap.String("instance_id", input.InstanceID),
		zap.String("previous_class", currentClass),
		zap.String("new_class", newClass),
		zap.String("status", newStatus),
		zap.Bool("apply_immediately", input.ApplyImmediately),
	)

	applyTime := "next maintenance window"
	if input.ApplyImmediately {
		applyTime = "immediately"
	}

	return s.successResult(result, startTime, fmt.Sprintf("Resize initiated, will apply %s", applyTime), map[string]interface{}{
		"previous_class": currentClass,
		"new_class":      newClass,
		"status":         newStatus,
		"apply_time":     applyTime,
	}), nil
}

// Helper methods

func (s *ActionService) successResult(result *ActionResult, startTime time.Time, message string, metadata map[string]interface{}) *ActionResult {
	endTime := time.Now()
	result.Status = "success"
	result.EndTime = &endTime
	result.Duration = endTime.Sub(startTime)
	result.Message = message
	if metadata != nil {
		if result.Metadata == nil {
			result.Metadata = metadata
		} else {
			for k, v := range metadata {
				result.Metadata[k] = v
			}
		}
	}
	return result
}

func (s *ActionService) failResult(result *ActionResult, startTime time.Time, message string, err error) *ActionResult {
	endTime := time.Now()
	result.Status = "failed"
	result.EndTime = &endTime
	result.Duration = endTime.Sub(startTime)
	result.Message = message
	if err != nil {
		result.Error = err.Error()
	}
	return result
}
