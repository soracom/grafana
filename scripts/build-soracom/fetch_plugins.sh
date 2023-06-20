cd "$(dirname "$0")"

#install this package in a throwaway dir so we can reuse it a few times
npm install --prefix ./local  @grafana/sign-plugin@latest -g
PLUGIN_DIR=./plugins

mkdir -p $PLUGIN_DIR
cd $PLUGIN_DIR

clone_private_repo () {
  if [ ! -z "$2" ]; then
    BRANCH=$2
  else 
    BRANCH=master
  fi
  
  if [ -d $1 ]; then
    cd $1
    git pull origin $BRANCH
    cd ..
  else
    echo "$(pwd)"
    echo `ls ..`
    echo `ls ../deploy_keys`
    echo `ls ../deploy_keys/$1/`
    echo "git clone --depth 1 --single-branch --branch $BRANCH git@github.com:soracom/$1.git $1"
    GIT_SSH_COMMAND="ssh -i ../deploy_keys/$1/id_rsa -F /dev/null" git clone --depth 1 --single-branch --branch $BRANCH git@github.com:soracom/$1.git $1
  fi

  if [ ! -z $1/signplugin.sh ]; then 
    cd $1
    ./signplugin.sh || exit 1
    cd ..
  fi
}

clone_public_repo () {
  if [ -d "$1-$2" ]; then
    cd $1-$2
    git pull origin master
    cd ..
  else
    git clone https://github.com/$1/$2.git $1-$2
  fi
  if [ ! -z "$3" ]; then
    cd $1-$2
    echo "Checking out specific commit $3"
    git checkout $3
    cd ..
  fi
}

build_repo () {
  if [ -d "$1-$2" ]; then
    cd $1-$2
    npm install
    npm run-script build
    cd ..
  fi
}

yarn_build_repo () {
  if [ -d "$1-$2" ]; then
    cd $1-$2
    npm install yarn
    npx yarn install --pure-lockfile
    npx yarn build
    cd ..
  fi
}

clone_private_repo soracom-harvest-backend main
clone_private_repo soracom-map-panel lagoon3-master
clone_private_repo soracom-image-panel lagoon3-master
clone_private_repo soracom-plot-panel main
clone_private_repo soracom-dynamic-image-panel lagoon3-master

#Add any pre-built plugins to the dir
cp -R ../pre-built-plugins/* .