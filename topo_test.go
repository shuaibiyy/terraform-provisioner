package main

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

const sampleConfig = `
tf_repo: https://github.com/shuaibiyy/ecs-jenkins.git

provisions:

  jenkins_1:
    action: apply
    parameters:
      desired_service_count: 3
      desired_instance_capacity: 2
      max_instance_size: 2
`

func TestConfigUnmarshalling(t *testing.T) {
    expected := Config{
        TfRepo: "https://github.com/shuaibiyy/ecs-jenkins.git",
        Provisions: map[string]Provision{
            "jenkins_1": Provision{
                Action: "apply",
                Parameters: map[string]string{
                    "desired_service_count": "3",
                    "desired_instance_capacity": "2",
                    "max_instance_size": "2",
                },
            },
        },
    }
    actual := getConfig(sampleConfig)

    assert.Equal(t, expected, actual)
}
