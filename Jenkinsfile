pipeline {
  agent any

  environment {
    IMAGE_TAG = "${env.BRANCH_NAME == 'master' ? '0.1' : '0.1-rc'}"
    ENV_FILE = "${env.BRANCH_NAME == 'master' ? 'prod.env' : 'staging.env'}"
    CONTAINER_NAME = "${env.BRANCH_NAME == 'master' ? 'mailman' : 'mailman-rc'}"
    KEY_DIR = "${env.BRANCH_NAME == 'master' ? 'prod' : 'staging'}"
  }

  stages {
    stage('Build') {
      steps {
        sh 'docker build -t mailman:$IMAGE_TAG .'
      }
    }

    stage('Deploy') {
      steps {
        // Don't fail the build if the container does not exist
        sh 'docker stop $CONTAINER_NAME || true'
        sh 'docker rm $CONTAINER_NAME || true'
        sh '''
          docker run -d \
            --restart always\
            --log-opt max-size=10m --log-opt max-file=3 \
            --name $CONTAINER_NAME \
            --env-file /home/env/mailman/$ENV_FILE \
            --network=ec2-user_default \
            mailman:$IMAGE_TAG
        '''
      }
    }
  }
}
