# Topo

Topo manages multiple provisions of the same [Terraform](https://terraform.io) scripts.
Topo clones a Terraform project specified in a configuration file, and runs parameterized Terraform commands on it.
Topo uses [Goroutines](https://www.golang-book.com/books/intro/10) to run Terraform commands and perform certain tasks concurrently. This helps Topo achieve acceptable speed.

Topo configuration file format:

    tf_repo: <git_repo_url>
    s3_bucket: <s3_bucket> # s3 bucket where Terraform remote state resides.
    
    provisions:
        <name>
            action: apply | destroy
            state: applied | changed | destroyed | nil
            parameters:
                <key>: <value>

## Usage

1. Create a yaml file using the Topo config format (you can refer to `topograph-sample.yml`), and name it something like `topograph.yml`.
2. Export the following environment variables:

        $ export AWS_ACCESS_KEY_ID="accesskey" # For tf to access AWS.
        $ export AWS_SECRET_ACCESS_KEY="secretkey"
        $ export AWS_DEFAULT_REGION="us-east-1"
        $ export TF_VAR_access_key=$AWS_ACCESS_KEY # Not necessary if the variable is not defined in your tf project.
        $ export TF_VAR_secret_key=$AWS_SECRET_ACCESS_KEY # Not necessary if the variable is not defined in your tf project.
        $ export TP_GIT_USER=<git_username> # Git credentials if tf project is in a private repository.
        $ export TP_GIT_PASSWORD=<git_password>
3. Run Topo with config created in step 1:

        $ topo [flags...] topograph.yml
    Available flags are:
    * `-update`: when specified guarantees that the terraform project will be fetched from remote, even if it was previously fetched. Note that flags must appear before the name of the config file, e.g.:
        
            $ topo -update <config_file>

## What exactly does Topo do?

1. Accepts and parses a YAML configuration file. A configuration file should contain one or more provision blocks, which look like:

        provisions:
          jenkins_2:
            action: apply
            state: changed
            parameters:
              desired_service_count: 1
              desired_instance_capacity: 1
              max_instance_size: 1
2. Each provision should have an action and/or state. A state may have the value `applied`, `destroyed`, or `changed`.
    An action may be either `apply` or `destroy`. The default action is `apply` and there is no default state. The provision will be ignored if any of the following cases are true:
    - `changed` state with a `destroy` action.
    - `destroyed` state with a `destroy` action.
    - `applied` state with an `apply` action.
3. Topo runs a parameterized terraform (tf) command on all provisions in the config based on their action and optional state.
4. A topo run involves the following:
    1. Cloning a git repo that contains tf scripts.
    2. Configuring the tf remote state.
    3. Running a tf command if none of the ignore criteria is met.
5. For each successful tf command, the provision's state in the config file is updated to either `applied` or `destroyed`.
