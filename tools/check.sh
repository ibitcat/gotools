#!/bin/bash

arr=( "w" "m" "g" "d" "l" )
path="$HOME/server/l-src/mmo"

echo ""
echo "-------- check process ----------"

for i in ${arr[@]}
do
	count=0
	pwdstr=""
	#找pid，可以有多个pid
	for pid in `ps -ef|grep "mmo $i" |awk '{print $2}'`
	do
		if [ -d /proc/${pid} ]; then #判断路径是否存在
			pwdstr=`ls -l /proc/${pid}/exe | awk '{print $11}'`

			if [ "$pwdstr" == "$path" ]; then
				let count+=1;
				echo "[$i] = OK, [pwd]=$pwdstr"
				#break 1
			fi
		fi
	done
done

echo "---- end check ----"
echo ""

