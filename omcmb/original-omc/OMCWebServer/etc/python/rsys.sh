#!/bin/bash
conf_file="/etc/rsyslog.conf"
#ip_port="1.1.1.1:8888"
ip_port=`docker exec -i redis redis-cli -a baiOMC@123 -p 6380 hget rsys_config ipPort`
type=`docker exec -i redis redis-cli -a baiOMC@123 -p 6380 hget rsys_config type`
result=`docker exec -i redis redis-cli -a baiOMC@123 -p 6380 hget rsys_config result`


function config_rsyslog(){
	sed -i '/^*.*@@'/d $conf_file
	sed -i '/^$ModLoad imuxsock'/d $conf_file
	sed -i '/^$ModLoad imjournal'/d $conf_file
	sed -i '/^$ModLoad imtcp'/d  $conf_file
	sed -i '/^$InputTCPServerRun'/d  $conf_file
	sed -i '/^$UDPServerRun'/d  $conf_file
	sed -i '/^$ModLoad imudp'/d  $conf_file

	echo '$ModLoad imuxsock' >> $conf_file
	echo '$ModLoad imjournal' >> $conf_file 
	echo '$ModLoad imtcp' >> $conf_file 
	echo '$InputTCPServerRun 514' >> $conf_file
	echo "*.* @@$ip_port" >> $conf_file

	systemctl restart rsyslog
	docker exec -i redis redis-cli -a baiOMC@123 -p 6380 hmset rsys_config result done
}

function disable_rsyslog(){
	sed -i '/^*.*@@'/d  $conf_file
	systemctl restart rsyslog
	docker exec -i redis redis-cli -a baiOMC@123 -p 6380 hmset rsys_config result done
}

LOCK_NAME="/tmp/rsys.lock"
    if ( set -o noclobber; echo "$$" > "$LOCK_NAME") 2> /dev/null;
    then
        trap 'rm -f "$LOCK_NAME"; exit $?' INT TERM EXIT
	
	if [ "$type" == "enable" -a "$result" == "start" ];then
		config_rsyslog
	elif [ "$type" == "disable" -a "$result" == "start" ];then
		disable_rsyslog
	      
	fi

        rm -f $LOCK_NAME
        trap - INT TERM EXIT
     else
        exit 1
     fi