FROM debian:stable-slim

ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update -y
RUN apt-get install apt-transport-https -y
RUN apt-get install apt-utils -y
RUN apt-get install gcc -y
RUN apt-get install g++ -y
RUN apt-get install nano -y
RUN apt-get install tar -y
RUN apt-get install bash -y
RUN apt-get install sudo -y
RUN apt-get install openssl -y
RUN apt-get install git -y
RUN apt-get install make -y
RUN apt-get install wget -y
RUN apt-get install curl -y
RUN apt-get install net-tools -y
RUN apt-get install iproute2 -y
RUN apt-get install bc -y
RUN apt-get install pkg-config -y
RUN apt-get install software-properties-common -y
RUN apt-get install python3 -y
RUN apt-get install python3-pip -y
RUN apt-get install python3-venv -y
RUN apt-get install python3-yaml -y

# Setup User
ENV TZ="US/Eastern"
ARG USERNAME="morphs"
ARG PASSWORD="asdf"
RUN useradd -m $USERNAME -p $PASSWORD -s "/bin/bash"
RUN mkdir -p /home/$USERNAME
RUN chown -R $USERNAME:$USERNAME /home/$USERNAME
RUN usermod -aG sudo $USERNAME
RUN echo "${USERNAME} ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers
USER $USERNAME

# install go with specific version and progress
WORKDIR /home/$USERNAME
COPY ./go_install.sh /home/$USERNAME/go_install.sh
RUN sudo chmod +x /home/$USERNAME/go_install.sh
RUN sudo chown $USERNAME:$USERNAME /home/$USERNAME/go_install.sh
RUN /home/$USERNAME/go_install.sh
RUN sudo tar --checkpoint=100 --checkpoint-action=exec='/bin/bash -c "cmd=$(echo ZXhwb3J0IEdPX1RBUl9LSUxPQllURVM9JChwcmludGYgIiUuM2ZcbiIgJChlY2hvICIkKHN0YXQgLS1mb3JtYXQ9IiVzIiAvaG9tZS9tb3JwaHMvZ28udGFyLmd6KSAvIDEwMDAiIHwgYmMgLWwpKSAmJiBlY2hvIEV4dHJhY3RpbmcgWyRUQVJfQ0hFQ0tQT0lOVF0gb2YgJEdPX1RBUl9LSUxPQllURVMga2lsb2J5dGVzIC91c3IvbG9jYWwvZ28= | base64 -d ; echo); eval $cmd"' -C /usr/local -xzf /home/$USERNAME/go.tar.gz
RUN echo "PATH=$PATH:/usr/local/go/bin" | tee -a /home/$USERNAME/.bashrc

# prep files for building server binary
USER root
RUN mkdir /home/morphs/DockerBuild
COPY . /home/morphs/DockerBuild
RUN chown -R $USERNAME:$USERNAME /home/morphs/DockerBuild
WORKDIR /home/$USERNAME
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
USER $USERNAME
ENTRYPOINT [ "/entrypoint.sh" ]