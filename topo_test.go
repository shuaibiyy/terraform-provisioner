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
		TfRepo: "https://github.com/shuaibiyy/ecs-jenkins.git",
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

func TestGetQualifiedConfig(t *testing.T) {
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
      desired_service_count: 1
      desired_instance_capacity: 1
      max_instance_size: 2

`

	expected := Config{
		TfRepo: "https://github.com/shuaibiyy/ecs-jenkins.git",
		S3Bucket: "bucket-topo",
		Provisions: map[string]Provision{
			"jenkins_2": Provision{
				Action: "apply",
				State: "changed",
				Parameters: map[string]string{
					"desired_service_count":     "1",
					"desired_instance_capacity": "1",
					"max_instance_size":         "2",
				},
			},
		},
	}

	config := getConfig(configYaml)
	actual := computeQualifiedConfig(&config)

	assert.Equal(t, expected, *actual)
}
