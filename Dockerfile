FROM kubefirst/kubefirst-builder:0.1-ubuntu

ADD nebulous /scripts/nebulous
ADD terraform /terraform

CMD [ "/bin/bash" ]
