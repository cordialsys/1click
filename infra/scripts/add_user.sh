#!/bin/bash
# this script is used to provision a user

set -ex
user=$1

if [ ! -f /tmp/.zshrc ]; then
    wget -O /tmp/.zshrc https://git.grml.org/f/grml-etc-core/etc/zsh/zshrc
fi

echo $user HOME is ~

mkdir -p ~/.docker

if grep "${user}" /etc/passwd >/dev/null 2>&1; then
    echo already added $user user
    chsh -s /bin/zsh $user
else
    useradd -m -G docker,systemd-journal -s /usr/bin/zsh ${user}
fi

# bootable containers recommand putting home dir in /var so it doesn't get deleted after update
# https://docs.fedoraproject.org/en-US/bootc/filesystem/#_filesystem_bind_mount_var
if [ ${user} != "root" ] ; then
    newHome="/var/home/${user}"
    mkdir -p $newHome
    chown -R "${user}" $newHome
    sudo usermod -d $newHome "${user}"
fi

sudo -i -u ${user} bash -c 'mkdir -p ~/.docker' 
sudo -i -u ${user} cp /tmp/.zshrc .

# add panel completions
panel completion bash > /etc/bash_completion.d/_panel
sudo -i -u ${user} bash -c '
mkdir -p .zsh/completions;
panel completion zsh > ~/.zsh/completions/_panel;
echo ". ~/.zsh/completions/_panel" >> ~/.zshrc;
echo ". /etc/bash_completion.d/_panel" >> ~/.bashrc;
'
