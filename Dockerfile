FROM kubefirst/kubefirst-builder:0.1-rc1

ADD nebulous /scripts/nebulous
ADD terraform /terraform

CMD [ "/bin/bash" ]
