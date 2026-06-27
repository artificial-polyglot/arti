# Arti Installtion Notes

Caveate: These are notes for installing Arti on RunPod.ai

## RunPod instructions

These are instructions for setting up a single node of Arti on a RunPod instance.

- Create RunPod Account
- go to console.runpod.io
- click Deploy Pod
- At Compute: Select CPU (best for installation)
- At Workload: Select Ubuntu 22.04
- Pod name: arti2
- Uncheck Start Jupyter Notebook
- Check SSH Terminal Access
- Enter public key
- At Compute: Network volume -> Create network volume

Selecting a region is an important decision, because one must have pods and storage in the same region.  Review the availability of inexpensive servers in various regions in order to select a region.  Also, select a region that supports S3, and CPU
      Some interesting servers are as follows:
      A40 $0.44
      L4 $0.39
      RTX 4090 $0.69
      RTX 5090 $0.99

- Data Center: Select US-CA-2
- Name: Enter arti2-disk
- Size: Set to 64 GB (might be too small)
- Click Create Network Volume
- At Region: Select the region chosen for arti2-disk
- At Compute: Select CPU, 4 vCPUs (for start)
- At Storage: change container disk to 10GB
- Click Deploy Pod

- It is AMD

- ssh root@38.80.152.147 -p 33976 -i ~/.ssh/id_rsa

- sudo apt -y update
- sudo apt -y upgrade

- Install miniconda or miniforge
- curl -LO https://github.com/conda-forge/miniforge/releases/latest/download/Miniforge3-Linux-x86_64.sh
- bash Miniforge3-Linux-x86_64.sh -b -p /workspace/miniforge3
- /workspace/miniforge3/bin/conda init bash
- source ~/.bashrc
- which conda

# Install executables using conda into the base environment
- conda activate base
- conda install -y ffmpeg -c conda-forge
- conda install -y sox -c conda-forge
- conda install -y sqlite -c conda-forge

- install go 
  - cd /workspace
  - https://go.dev/doc/install (to check for latest version)
  - curl -LO https://go.dev/dl/go1.26.4.linux-amd64.tar.gz
  - mkdir /workspace/local
  - mv go1.26.4.linux-amd64.tar.gz /workspace/local
  - cd /workspace/local
  - tar -xzf go1.26.4.linux-amd64.tar.gz

- git clone https://github.com/artificial-polyglot/arti.git
- export GOROOT=/workspace/local/go
- export GOPATH=/workspace/gopath
- export PATH="$GOROOT/bin:$GOPATH/bin:$PATH"
- cd src/arti
- go mod download
- go install controller/dataset_cli (this should be renamed)
- cd encode
- sh build_aeneas_env.sh (aeneas env)
- cd ../mms/adapter
- sh build_env.sh (mms_adapter env)
- cd ../../mms/mms_align 
- sh build_env.sh (mms_fa env) 
- cd ../../mms/mms_asr
- sh build_env.sh (mms_asr env)
- cd ../../speech_to_text
- whisper.go contains installation notes

- cd /workplace
- mkdir data
- mkdir data/download
- mkdir data/tmp
- 
- copy runpod_dev.env to /workspace
- copy runpod_dev.secret to /workspace
- edit /root/.bashrc source these two files.

fasttext not included


-
