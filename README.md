# Topograph

[WORK IN PROGRESS]

A project for managing multiple provisions of the same Terraform scripts.

## Usage

    $> topograph topograph.yml

## What does it do?

1. Accepts and parses a YAML configuration file. A configuration file should contain one or more provision blocks, which look like:

        provision:
          parameters:
            desired_service_count: 3
            desired_instance_capacity: 2
            max_instance_size: 2
          action: apply
2. Each provision should have an action and/or state. A state may have the value `applied`, `destroyed`, or `changed`.
    An action may be either `apply` or `destroy`. The default action is `apply` and there is no default state. The provision will be ignored if any of the following cases are true:
    - `changed` state with a `destroy` action.
    - `destroyed` state with a `destroy` action.
    - `applied` state with an `apply` action.
3. Topograph runs a parameterized terraform (tf) command on all provisions in the config based on their action and optional state.
4. A topograph run involves the following:
    1. Checking out a git repo.
    2. Configuring the tf remote state.
    3. Running a tf command if none of the ignore criteria is met.
5. For each successful tf command, the provision's state in the config file is updated to either `applied` or `destroyed`.
