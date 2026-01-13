#!/bin/sh

[ -z "$DDNS_DOMAINS" ] && echo "DDNS_DOMAINS not set" && exit 1
[ -z "$DDNS_PARENT_NS" ] && echo "DDNS_PARENT_NS not set" && exit 1

[ -z "$DDNS_DEFAULT_TTL" ] && DDNS_DEFAULT_TTL=3600
[ -z "$DDNS_TRANSFER" ] && DDNS_TRANSFER=none

if [ -z "$DDNS_IP" -a -z "$DDNS_IP6" ] ; then
        GUESS_IP="$(curl icanhazip.com)"
        case "$GUESS_IP" in
        *:*) DDNS_IP6="$GUESS_IP" ;;
        *.*) DDNS_IP="$GUESS_IP" ;;
        *)   echo "Failed to guess IP" && exit 1
        esac
        echo "Guessed own IP as $GUESS_IP"
fi

for d in ${DDNS_DOMAINS//,/ }
do
        if ! grep -sq 'zone "'$d'"' /etc/bind/named.conf.ddns
        then
                echo "creating zone...";
                cat >> /etc/bind/named.conf.ddns <<EOF
zone "$d" {
        type master;
        file "pri/$d.zone";
        allow-query { any; };
        allow-transfer { ${DDNS_TRANSFER}; };
        allow-update { localhost; };
        also-notify { ${DDNS_NOTIFY} };
};
EOF
        fi

        if [ ! -f /var/bind/pri/$d.zone ]
        then
                echo "creating zone file..."
                cat > /var/bind/pri/$d.zone <<EOF
\$ORIGIN .
\$TTL 86400     ; 1 day
$d              IN SOA  ${DDNS_PARENT_NS}. root.${d}. (
                                74         ; serial
                                3600       ; refresh (1 hour)
                                900        ; retry (15 minutes)
                                604800     ; expire (1 week)
                                86400      ; minimum (1 day)
                                )
                        NS      ${DDNS_PARENT_NS}.
                        ${DDNS_IP:+A       $DDNS_IP}
                        ${DDNS_IP6:+AAAA    $DDNS_IP6}
\$ORIGIN ${d}.
\$TTL ${DDNS_DEFAULT_TTL}
EOF
        fi
done

# hand over zone files to the bind user and allow creating jnl files
chown -R named:named /var/bind/pri