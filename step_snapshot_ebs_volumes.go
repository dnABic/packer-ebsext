package ebsext

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/builder/amazon/common"
	awscommon "github.com/mitchellh/packer/builder/amazon/common"
	"github.com/mitchellh/packer/packer"
	"github.com/mitchellh/packer/template/interpolate"
)

type stepSnapshotEBSVolumes struct {
	VolumeRunTags     map[string]string
	Ctx               interpolate.Context
	VolumeDoSnapshot  bool
	SnapshotEbsVolume []string
}

func matchDevice(deviceName string, ebsDeviceNames []string) bool {
	for _, name := range ebsDeviceNames {
		if name == deviceName {
			return true
		}
	}
	return false
}

func (s *stepSnapshotEBSVolumes) Run(state multistep.StateBag) multistep.StepAction {
	ec2conn := state.Get("ec2").(*ec2.EC2)
	instance := state.Get("instance").(*ec2.Instance)
	sourceAMI := state.Get("source_image").(*ec2.Image)
	ui := state.Get("ui").(packer.Ui)

	if s.VolumeDoSnapshot == false || len(s.SnapshotEbsVolume) == 0 {
		return multistep.ActionContinue
	}

	volumeIds := make([]*string, 0)
	for _, v := range instance.BlockDeviceMappings {
		ui.Say(fmt.Sprintf("DEBUG device %s", v.DeviceName))
		if ebs := v.Ebs; ebs != nil && matchDevice(v.DeviceName, SnapshotEbsVolume) {
			ui.Say(fmt.Sprintf("DEBUG preparing for snapshot device %s", v.DeviceName))
			volumeIds = append(volumeIds, ebs.VolumeId)
		}
	}

	if len(volumeIds) == 0 {
		return multistep.ActionContinue
	}

	var snapshotIds []*string
	for _, v := range volumeIds {
		createSnapResp, err := ec2conn.CreateSnapshot(&ec2.CreateSnapshotInput{
			VolumeId: v,
		})
		if err != nil {
			err := fmt.Errorf("Error while creating snapshot of EBS Volume %s: %s", v, err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		var snapshotIdp *string
		snapshotId := *createSnapResp.SnapshotId
		snapshotIdp = createSnapResp.SnapshotId
		snapshotIds = append(snapshotIds, snapshotIdp)
		ui.Say(fmt.Sprintf("Creating snapshot of volume %s with ID %s", v, snapshotId))

		stateChange := awscommon.StateChangeConf{
			Pending: []string{"pending"},
			Target:  "completed",
			Refresh: func() (interface{}, string, error) {
				resp, err := ec2conn.DescribeSnapshots(&ec2.DescribeSnapshotsInput{SnapshotIds: []*string{&snapshotId}})
				if err != nil {
					return nil, "", err
				}

				if len(resp.Snapshots) == 0 {
					return nil, "", errors.New("No snapshots found.")
				}

				s := resp.Snapshots[0]
				return s, *s.State, nil
				//return nil, "", nil
			},
			StepState: state,
		}

		ui.Say("Waiting for EBS Volume snapshot to complete...")
		if _, err := awscommon.WaitForState(&stateChange); err != nil {
			err := fmt.Errorf("Error waiting for EBS Volume snapshot: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

	}

	if len(s.VolumeRunTags) == 0 {
		return multistep.ActionContinue
	}

	ui.Say("Adding tags to EBS Volumes Snapshots")
	tags, err := common.ConvertToEC2Tags(s.VolumeRunTags, *ec2conn.Config.Region, *sourceAMI.ImageId, s.Ctx)
	if err != nil {
		err := fmt.Errorf("Error tagging EBS Volumes Snapshots: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	_, err = ec2conn.CreateTags(&ec2.CreateTagsInput{
		Resources: snapshotIds,
		Tags:      tags,
	})

	if err != nil {
		err := fmt.Errorf("Error tagging EBS Volume Snapshots: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *stepSnapshotEBSVolumes) Cleanup(state multistep.StateBag) {
	// No cleanup...
}
