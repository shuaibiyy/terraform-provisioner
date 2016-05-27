package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigUnmarshalling(t *testing.T) {
	const configYaml = `
tf_repo: https://github.com/shuaibiyy/ecs-jenkins.git
s3_bucket: bucket-topo

provisions:

  jenkins_1:
    action: apply
    parameters:
      desired_service_count: 3
      desired_instance_capacity: 2
      max_instance_size: 2
`

	expected := Config{
		TfRepo:   "https://github.com/shuaibiyy/ecs-jenkins.git",
		S3Bucket: "bucket-topo",
		Provisions: map[string]Provision{
			"jenkins_1": Provision{
				Action: "apply",
				Parameters: map[string]string{
					"desired_service_count":     "3",
					"desired_instance_capacity": "2",
					"max_instance_size":         "2",
				},
			},
		},
	}
	actual := getConfig(configYaml)
	assert.Equal(t, expected, actual)
}

func TestComputeQualifiedConfig(t *testing.T) {
	const configYaml = `
tf_repo: https://github.com/shuaibiyy/ecs-jenkins.git
s3_bucket: bucket-topo

provisions:

  jenkins_1:
    action: destroy
    state: destroyed
    parameters:
      desired_service_count: 3
      desired_instance_capacity: 2
      max_instance_size: 2

  jenkins_2:
    action: apply
    state: changed
    parameters:
      desired_service_count: 5
      desired_instance_capacity: 3
      max_instance_size: 4

  jenkins_3:
    action: destroy
    state: changed
    parameters:
      desired_service_count: 1
      desired_instance_capacity: 1
      max_instance_size: 2

  jenkins_4:
    action: destroy
    state: applied
    parameters:
      desired_service_count: 2
      desired_instance_capacity: 2
      max_instance_size: 2

  jenkins_5:
    action: apply
    state: applied
    parameters:
      desired_service_count: 4
      desired_instance_capacity: 2
      max_instance_size: 3
`

	expected := Config{
		TfRepo:   "https://github.com/shuaibiyy/ecs-jenkins.git",
		S3Bucket: "bucket-topo",
		Provisions: map[string]Provision{
			"jenkins_2": Provision{
				Action: "apply",
				State:  "changed",
				Parameters: map[string]string{
					"desired_service_count":     "5",
					"desired_instance_capacity": "3",
					"max_instance_size":         "4",
				},
			},
			"jenkins_4": Provision{
				Action: "destroy",
				State:  "applied",
				Parameters: map[string]string{
					"desired_service_count":     "2",
					"desired_instance_capacity": "2",
					"max_instance_size":         "2",
				},
			},
		},
	}

	config := getConfig(configYaml)
	actual := computeQualifiedConfig(&config)
	assert.Equal(t, expected, *actual)
}

func TestPrepareDestroyArgs(t *testing.T) {
	var expected []interface{}
	expected = append(expected, "destroy", "-force", "-var", "desired_service_count=2",
		"-var", "desired_instance_capacity=1", "-var", "max_instance_size=1")
	c := Config{
		TfRepo:   "https://github.com/shuaibiyy/ecs-jenkins.git",
		S3Bucket: "bucket-topo",
		Provisions: map[string]Provision{
			"jenkins_2": Provision{
				Action: "destroy",
				State:  "applied",
				Parameters: map[string]string{
					"desired_service_count":     "2",
					"desired_instance_capacity": "1",
					"max_instance_size":         "1",
				},
			},
		},
	}
	p := c.Provisions["jenkins_2"]
	actual := prepareDestroyArgs(&p)
	assert.Equal(t, expected, actual)
}

func TestPrepareApplyArgs(t *testing.T) {
	var expected []interface{}
	expected = append(expected, "apply", "-var", "desired_service_count=2",
		"-var", "desired_instance_capacity=1", "-var", "max_instance_size=1")
	c := Config{
		TfRepo:   "https://github.com/shuaibiyy/ecs-jenkins.git",
		S3Bucket: "bucket-topo",
		Provisions: map[string]Provision{
			"jenkins_2": Provision{
				Action: "apply",
				State:  "changed",
				Parameters: map[string]string{
					"desired_service_count":     "2",
					"desired_instance_capacity": "1",
					"max_instance_size":         "1",
				},
			},
		},
	}
	p := c.Provisions["jenkins_2"]
	actual := prepareApplyArgs(&p)
	assert.Equal(t, expected, actual)
}
