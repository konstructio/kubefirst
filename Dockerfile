FROM kubefirst/kubefirst-builder:0.1-ubuntu

ADD scripts/nebulous /scripts/nebulous
ADD terraform /terraform

RUN apt-get update
RUN apt-get install dnsutils -y

CMD [ "/bin/bash" ]
