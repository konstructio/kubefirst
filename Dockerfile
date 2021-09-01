FROM kubefirst/kubefirst-builder:0.4-ubuntu

ADD scripts/nebulous /scripts/nebulous
ADD gitops /gitops

RUN apt-get update
RUN apt-get install dnsutils -y

CMD [ "/bin/bash" ]
