#!/usr/bin/groovy

@Library(['com.optum.jenkins.pipelines.templates.terraform.fork.cc-team@master']) _

def lockName = "",
    namespace = "",
    deploymentRing = "PR",
    optumfile

// only execute pull requests and master branch
if (env.CHANGE_ID == null && env.BRANCH_NAME != "master") {
    return
}

node("hcc-build-docker") {
    checkout scm

    optumfile = readYaml file: "Optumfile.yml"

    if (env.CHANGE_ID) {
      namespace = "pr-${env.CHANGE_ID}"
      lockName = "${env.JOB_NAME}/pr-${env.CHANGE_ID}"
    } else {
      lockName = "${env.JOB_NAME}/pr"
      deploymentRing = "INTERNAL"
    }

    // deployment to pr account (may or may not have namespace between pr and master)
    handleDeployment(optumfile, "346166872260", namespace, lockName, deploymentRing)

    // if master, handle prod deployment
    if (env.BRANCH_NAME == "master") {
        stage('Prod') {
            lockName = "${env.JOB_NAME}/prod"
            deploymentRing = "PROD"
            handleDeployment(optumfile, "626017279283", namespace, lockName, deploymentRing)
        }
    }
 }



def withAssumedAccessSh(String account, Closure code) {
    try {
        withCredentials([
                    string(credentialsId: "MASTERADMINKEY", variable: 'AWS_MASTER_KEY'),
                    string(credentialsId: "MASTERADMINSECRET", variable: 'AWS_MASTER_SECRET')])
        {
            assert AWS_MASTER_KEY : "AWS_MASTER_KEY must have value"
            assert AWS_MASTER_SECRET : "AWS_MASTER_SECRET must have value"

            sh """
            set +x
            export AWS_ACCESS_KEY_ID=${AWS_MASTER_KEY} &>/dev/null
            export AWS_SECRET_ACCESS_KEY=${AWS_MASTER_SECRET} &>/dev/null
            creds=`aws sts assume-role --role-arn arn:aws:iam::${account}:role/TerrascaleDeployRole --role-session-name Terrascale-Deploy-${account} --output json | jq '{ accessKeyId: .Credentials.AccessKeyId, secretAccessKey: .Credentials.SecretAccessKey, sessionToken: .Credentials.SessionToken }'` &>/dev/null
            export AWS_ACCESS_KEY_ID=\$(echo \$creds | jq -r '.accessKeyId') &>/dev/null
            export AWS_SECRET_ACCESS_KEY=\$(echo \$creds | jq -r '.secretAccessKey' ) &>/dev/null
            export AWS_SESSION_TOKEN=\$(echo \$creds | jq -r '.sessionToken') &>/dev/null
            set -x
            ${code()}
            """
        }
    } catch (Exception ex) {
        echo "Exception: ${ex}"
        throw ex
    }
}

/**
 * Helper function execute deployment
 */
def handleDeployment(optumfile, String account, String namespace, String lockName, String deploymentRing) {

    updateBuildStatus(optumfile, "pending", "deployment", "Deployment is running")

    lock(resource: lockName) {
        stage('Deploy Containers') {
             try {
                withAssumedAccessSh(account) {
                """
                if [ -z "${namespace}" ]; then
                    bash deploy-containers.sh --account-id ${account} --aws-profile "static"
                else
                    bash deploy-containers.sh --account-id ${account} --aws-profile "static" --namespace ${namespace}
                fi
                """
                }

                dir('reports') {
                    junit "junit.xml"
                    if (currentBuild.result == "UNSTABLE") {
                        updateBuildStatus(optumfile, "error", "unit-tests", "Unit tests failed")
                    } else {
                        updateBuildStatus(optumfile, "success", "unit-tests", "Unit tests passed")
                    }
                }

            } catch(Exception e) {
               echo "Exception caught when deploying containers: ${e}"
               updateBuildStatus(optumfile, "error", "deployment", "Containers deployment failed")
               throw e
            }
        }
        milestone()
        updateBuildStatus(optumfile, "success", "deployment", "Deployment is complete")
    }
}

