FROM hyperledger/fabric-peer:x86_64-1.0.0-rc1

ENV FABRIC_CFG_PATH /etc/hyperledger/fabric
RUN mkdir -p $FABRIC_CFG_PATH

COPY ./bin/* /usr/local/bin/
COPY ./scripts/* /usr/local/bin/
COPY ./payload/snapsd /usr/local/bin/

ADD payload/sampleconfig $FABRIC_CFG_PATH

EXPOSE 7051 7053

ENTRYPOINT ["/usr/local/bin/dumb-init", "--"]
CMD ["/usr/local/bin/entrypoint.sh"]
