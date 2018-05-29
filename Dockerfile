FROM alpine:3.5

RUN apk update && \
    apk add ca-certificates bash
    
WORKDIR /bin

ADD ./k8s-azurekeyvault-sidecar /bin/k8s-azurekeyvault-sidecar

CMD ["/bin/k8s-azurekeyvault-sidecar", "-logtostderr=1"] 