#!/bin/bash -v

runuser --login ec2-user --shell=/bin/bash << 'EOF'
env
cd ~/go/src/fcbh-dataset-io
if [[ "$FCBH_DATASET_QUEUE" == *"-dev" ]]; then
    #git pull origin train
    git pull origin main
else
    git pull origin main
fi
go install -a ./controller/queue_server
cp /home/ec2-user/go/bin/queue_server /var/app/current/queue_server
cd
#nohup ~/go/bin/queue_server &
sudo systemctl restart queue_server.service
EOF
exit 0
