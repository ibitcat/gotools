#!/bin/bash

cd ./l-src

#vertical="$1"
vertical="v"
if [ $vertical == "v" ]; then
	tmux new-session -s server -n w-m-g-d-t -d './mmo w 1'
	echo "w start!"
	sleep 0.5

	tmux split-window -h -d './mmo m 1'
	echo "m start!" 
	sleep 0.5

	tmux split-window -v -p 25 './mmo g 1'
	tmux select-pane -t 1
	echo "g start!"
	sleep 0.5

	tmux split-window -v -d './mmo d 1'
	echo "d start!"
	tmux select-pane -R 
	sleep 0.5
	
	tmux split-window -v -p 25 -d './mmo l 1'
	echo "l start!"
	tmux select-pane -D 
	sleep 0.5

	#tmux split-window -v -p 25 -d 'telnet 0.0.0.0 5002'
	tmux splitw -v -d './cmd_tool'
	tmux select-pane -D 
	echo "t start!"
else
	tmux new-session -s server -n w-m-g-d-t -d './mmo w 1'
	echo "w start!"
	sleep 0.5

	tmux split-window -h -d './mmo m 1'
	echo "m start!"
	tmux select-pane -R 
	sleep 0.5

	tmux split-window -h -d './mmo g 1'
	echo "g start!"
	tmux select-pane -R
	tmux select-layout even-horizontal
	sleep 0.5

	tmux split-window -v -p 66 -d './mmo d 1'
	echo "d start!"
	tmux select-pane -D 
	sleep 0.5

	#tmux split-window -v -p 50 -d 'telnet 0.0.0.0 5002'
	tmux splitw -v -p 50 -d './cmd_tool'
	tmux select-pane -D 
	echo "t start!"
fi


#windowname=""
#first=1
#session=server[$*]
#
#insert() 
#{
#	eval "$1=\"$2\""
#}
#
#get_value() 
#{
#	eval "echo \$$1"
#}
#
#
#doTmux()
#{
#	if [ "$1" != "t" ]; then
#		if [ $first == 1 ]; then
#			windowname="$1"
#			tmux new-session -s "$session" -n w-m-g-d-t -d './mmo' "$1" 1
#			first=0
#		else
#			windowname="$windowname-$1"
#			tmux split-window -h -d './mmo' "$1" 1
#			tmux select-pane -R
#		fi
#	else
#		if [ $first == 1 ]; then
#			windowname="$1"
#			tmux new-session -s "$session" -n w-m-g-d-t -d 'telnet 0.0.0.0 5002'
#			first=0
#		else
#			windowname="$windowname-$1"
#			tmux split-window -h -d 'telnet 0.0.0.0 5002'
#			tmux select-pane -R 
#		fi
#	fi
#
#	echo "$1 start!"
#	sleep 0.5
#}
#
#
#exclude=0
#arr=( "w" "m" "g" "d" "t" )
#arg=(m t)
#
#if [ "$1" ]; then
#	if [ "$1" == "-" ]; then
#		exclude=1
#	fi
#fi
#
#for i in ${arr[@]}
#do
#	if [ $# == 0 ]; then
#		doTmux $i
#	elif [ $exclude == 1 ]; then
#		isExc=0
#		for args in $@
#		do
#			if [ "$i" == "$args" ]; then
#				isExc=1
#				break
#			fi
#		done
#		if [ $isExc != 1 ]; then
#			doTmux $i
#		fi
#	else
#		for args in $@
#		do
#			if [ "$i" == "$args" ]; then
#				doTmux $i
#				break
#			fi
#		done
#	fi
#done
#
#tmux renamew $windowname
#tmux select-layout even-horizontal


#tmux new-session -s server -n w-m-g-d-t -d './mmo w 1'
#tmux split-window -h -p 80 -d './mmo m 1' #-p 80表示80%宽度
#tmux select-pane -t 0
#tmux select-pane -R
#tmux split-window -h -d './mmo g 1'
#tmux selectp -t 0
#tmux select-pane -R
#tmux split-window -h -d './mmo l 1'
#tmux selectp -t 2
#tmux select-pane -R
#tmux split-window -h -d './mmo d 1'
#tmux select-layout even-horizontal
#tmux lastp  #选择上一个panel
#tmux select-pane -t 2
#tmux split-window -v -d 'telnet 0.0.0.0 5002'
#tmux select-pane -t 3
