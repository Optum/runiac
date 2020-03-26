#!/usr/bin/groovy

@Library(['com.optum.jenkins.pipelines.templates.terraform.fork.cc-team@master']) _

import com.optum.template.terraform.utils.TimeUtil


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
 }

// if master, handle prod deployment
if (env.BRANCH_NAME == "master") {
    stage('Prod') {
        timeout(time: 30, unit: "DAYS") {
            input "Proceed with deployments for PROD environment?"
        }

        node('hcc-build-docker') {
            checkout scm
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
            creds=`aws sts assume-role --role-arn arn:aws:iam::${account}:role/GaiaDeployRole --role-session-name Gaia-Deploy-${account} --output json | jq '{ accessKeyId: .Credentials.AccessKeyId, secretAccessKey: .Credentials.SecretAccessKey, sessionToken: .Credentials.SessionToken }'` &>/dev/null
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
 * Helper function execute teardown
 */
def handleDeployment(optumfile, String account, String namespace, String lockName, String deploymentRing) {

    updateBuildStatus(optumfile, "pending", "deployment", "Deployment is running")

    lock(resource: lockName) {
        stage('Deploy Containers') {
             try {
                dir('containers') {
                    withAssumedAccessSh(account) {
                    """
                    if [ -z "${namespace}" ]; then
                        bash deploy-containers.sh --account-id ${account} --aws-profile "static"
                    else
                        bash deploy-containers.sh --account-id ${account} --aws-profile "static" --namespace ${namespace}
                    fi
                    """
                    }

                    junit "**/*.xml"
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
        milestone(ordinal: 0)
        updateBuildStatus(optumfile, "success", "deployment", "Deployment is complete")
    }
}

/**
 * Helper function execute teardown
 */
def handleTeardown(optumfile, String account, String namespace, String lockName, String deploymentRing) {
    def destroyPromptTimeoutDays = 1 // 2 weeks would be better (equiv of one sprint) to address pain point of having to frequently rebuild our pr envs
    def readyForTeardown = true

    // Milestones are used here to prevent resources from being deleted when a prior build's timeout occurs.
    // The current build's number is used to enforce sequencing of builds, so the right one(s) are auto-aborted
    // Multiple milestones are needed because a milestone isn't seen as passed in Jenkins until a higher milestone is reached.  Without all these milestone triggers, prior
    //   builds won't abort when a new build reaches the same logical step.
    Integer buildNumber = (currentBuild.displayName).substring(1)
    def earlierMilestones = 1
    for (i = (0 + earlierMilestones); i <= (earlierMilestones + (buildNumber + 1)); i++) {
      milestone(ordinal: i)
    }

    updateBuildStatus(optumfile, "pending", "teardown", "Waiting at teardown prompt")

    readyForTeardown = false
    def resetCount = 0

    while (!readyForTeardown) {
        def userAbort = false
        def userSelection = "Teardown Now"
        def didTimeout = false
        def otherSystemAbortOccurred = false
        def submitter
        Date teardownTime = TimeUtil.plusDays(new Date(), destroyPromptTimeoutDays) // plusMinutes(new Date(), 1)

        try {
            echo "timeout will expire at ${teardownTime}"
            timeout(time: destroyPromptTimeoutDays, unit: 'DAYS') {
                Map responseValues = input(
                    id: "teardown_prompt",
                    message: "Teardown now or reset timer? (teardown will occur if 'abort' is clicked below or automatically if timer is not reset by ${teardownTime})",
                    parameters: [
                      choice(choices: "Teardown Now\nReset Timer", description: '', name: 'userSelection'),
                    ],
                    submitterParameter: 'submitter'
                )
              // if above throws no exception, then user made a selection and clicked proceed
              userAbort = false
              submitter = responseValues.submitter
              userSelection = responseValues.userSelection
            }
        }
        catch (err) // system aborted (due to timeout, milestone, redX, etc), or user-aborted at prompt
        {
            def user = err.getCauses()[0].getUser()
            if ('SYSTEM' != user.toString()) {  // user aborted at prompt
                userAbort = true
                echo "Aborted by: [${user}]"
                submitter = user.toString()
            } else {
                // user = SYSTEM can be the result of a timeout, milestone, red X, or other(?) abort
                // need to differentiate milestone from any others because milestone aborts should not result in teardown
                // unfortunately, Jenkins doesn't populate org.jenkinsci.plugins.workflow.support.steps.input.Rejection with anything that would help differentiate among
                //   the possible abort reasons for the abort types when user == SYSTEM.
                // We can at least differentiate timeout by doing a time comparison of now vs when the prompt was first presesented (or most recently reset).
                if (teardownTime.compareTo(new Date()) <= 0) {
                    didTimeout = true
                } else {
                    otherSystemAbortOccurred = true
                }
            }
        }

        if (didTimeout) {
            // prompt timed-out
            echo "Teardown timeout reached."
            readyForTeardown = true
        } else if (userAbort) {
            // user aborted at prompt
            echo "${submitter} aborted build at prompt."
            currentBuild.result = 'UNSTABLE'
            readyForTeardown = false
            break
        } else if (otherSystemAbortOccurred) {
            // milestone or red-X termination of build occurred.
            // exit the build immediately as an UNSTABLE build.
            echo "A non-timeout system abort occurred (milestone abort or red X kill).  No teardown will occur."
            currentBuild.result = 'UNSTABLE'
            readyForTeardown = false
            break
        } else {
            // user answered prompt.  Reset timer or teardown as specified by the user.
            if (userSelection == "Teardown Now") {
                echo "${submitter} chose to teardown."
                readyForTeardown = true
            } else if (userSelection == "Reset Timer") {
                resetCount++
                echo "Re-prompting for teardown.  Teardown must eventually occur, but can be deferred with continued user resets. (current reset count: ${resetCount})"
            } else {
                echo "Something unexpected happened.  Assuming that a reset was intended.  If you need to escape a loop of this occurring, choose another option or redX abort the build."
                resetCount++
                echo "Re-prompting for teardown.  Teardown must eventually occur, but can be deferred with continued user resets. (current reset count: ${resetCount})"
            }
        }
    }

    if (readyForTeardown) {
         node("hcc-build-docker") {

            lock(resource: lockName) {
                try {

                } catch(Exception e) {
                   echo "Exception caught during teardown: ${e}"
                   updateBuildStatus(optumfile, "error", "teardown", "Teardown failed")
                   throw e
                }
            }

          updateBuildStatus(optumfile, "success", "teardown", "Teardown complete")
        }
    }
}

