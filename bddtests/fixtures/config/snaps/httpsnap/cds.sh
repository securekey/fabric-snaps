#!/bin/bash

NAME=httpsnap
CDS=/opt/extsysccs/$NAME.golang
SOURCE=github.com/securekey/fabric-snaps/httpsnap/cmd

peer chaincode package -n $NAME -p $SOURCE -v 1.0.0 $CDS

chmod 775 $CDS
