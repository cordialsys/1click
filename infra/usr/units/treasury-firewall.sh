#!/bin/bash

# Any ports that could be exposed by treasury (8777, 26656, and 7867 are the public ports)
PORTS="26656,26657,1317,7867,7666"

# 8777 should be the only open port

# delete the rules
iptables -D DOCKER-USER -i lo -p tcp --match multiport --dport $PORTS -j ACCEPT || true
iptables -D DOCKER-USER -i wt0 -p tcp --match multiport --dports $PORTS -j ACCEPT || true
iptables -D DOCKER-USER -i docker0 -p tcp --match multiport --dports $PORTS -j ACCEPT || true
iptables -D DOCKER-USER -i wg0 -p tcp --match multiport --dports $PORTS -j ACCEPT || true
iptables -D DOCKER-USER -p tcp --match multiport --dport $PORTS -j DROP || true


if [[ $1 = "add" ]]; then

    echo "Adding treasury firewall rules"
    iptables -N DOCKER-USER  || true

    # permit from localhost
    iptables -I DOCKER-USER -i lo -p tcp --match multiport --dport $PORTS -j ACCEPT
    # permit from vpn interface (e.g. netbird)
    iptables -I DOCKER-USER -i wt0 -p tcp --match multiport --dports $PORTS -j ACCEPT
    # permit from docker0 interface
    iptables -I DOCKER-USER -i docker0 -p tcp --match multiport --dports $PORTS -j ACCEPT
    # permit from wireguard interface
    iptables -I DOCKER-USER -i wg0 -p tcp --match multiport --dports $PORTS -j ACCEPT
    # deny all else
    iptables -A DOCKER-USER -p tcp --match multiport --dport $PORTS -j DROP

else

    echo "Removed treasury firewall rules"

fi
