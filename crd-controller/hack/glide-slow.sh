    #!/bin/bash
# You can execute me through Glide by doing the following:
# - Execute `glide slow`
# - ???
# - Profit
pushd $GOPATH/src/github.com/kubernetes-practice/crd-controller 

glide up -v
glide vc --only-code --no-tests

popd