FROM kubefirst/kubefirst-builder:0.1-ubuntu

ADD scripts/nebulous /scripts/nebulous
ADD terraform /terraform

CMD [ "/bin/bash" ]
