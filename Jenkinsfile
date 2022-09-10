pipeline {
	agent { label 'bikenow-vm' }

    environment {
        DOCKER_IMAGE_URL = 'bikenow.vkw.tu-dresden.de/priobike/prediction-monitor'
        DOCKER_REGISTRY_URL = 'https://bikenow.vkw.tu-dresden.de/priobike'
        DOCKER_REGISTRY_CREDENTIALS_ID = 'docker-publisher'
        
        GIT_TAG_NAME = getGitTagName()
    }

    stages {
        stage('Build and Publish Containers') {
            steps {
                script {
                    // Simply tag with branch name
                    imageTags = [env.BRANCH_NAME]

                    // Tag with git tag, if available
                    if (env.GIT_TAG_NAME != null) {
                        echo "No git tag found, will not tag the docker image with a git tag"
                    } else {
                        echo "Found git tag: $env.GIT_TAG_NAME - will tag the docker image with this tag"
                        imageTags.add(env.GIT_TAG_NAME)
                    }

                    imageTags = imageTags.collect { sanitizeDockerTag(it) }

                    echo "Will create container tags: $imageTags"
                }

                // Build and tag docker image
                script {
                    imageTagsParameters = imageTags.collect { "-t ${env.DOCKER_IMAGE_URL}:$it" }.join(' ')
                    // Build the distributable backend image, because the frontend image is only needed for development
                    sh "docker build -f \"Dockerfile\" $imageTagsParameters ."
                }

                // Push docker image to registry
                withDockerRegistry(credentialsId: env.DOCKER_REGISTRY_CREDENTIALS_ID, url: env.DOCKER_REGISTRY_URL) {
                    script {
                        imageTags.each {
                            sh "docker push ${env.DOCKER_IMAGE_URL}:$it"
                        }
                    }
                }
            }
        }
    }

}

String sanitizeDockerTag(String tag) {
    tag.replaceAll(/[^a-zA-Z0-9\-\_\.]/, "-").replaceAll(/-+/, "-")
}

String getGitTagName() {
    commit = sh(script: 'git rev-parse HEAD', returnStdout: true)?.trim()
    if (commit) {
        scriptToRun = "git describe --tags ${commit}"
        hasTags = sh(script: scriptToRun, returnStatus: true) == 0
        if (!hasTags) {
            return null
        }

        return sh(script: scriptToRun, returnStdout: true)?.trim()
    }
    return null
}
