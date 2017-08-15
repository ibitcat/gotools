#!/bin/bash
#此脚本的主要用途是检测mysql服务器上所有的db或者单独db中的坏表
#变量说明 pass mysql账户口令 name mysql账号名称 data_path mysql目录路径 directory_list 目录列表 file_list文件列表 db_name 数据库名称 repair_count单库中待修复的表总数
#变量说明 repair_count_all所有库中待修复的表总数 mysql_version mysql版本 _file_name 数据表名称
#mysql存储路径: mysql数据库文件存储的路径，路径配置在my.cnf中的字段datadir=/var/lib/mysql
#mysql命令路径: mysql二进制路径，使用which mysql，可以查询

echo -e "此脚本的主要用途是检测mysql服务器上所有的数据库或者单独数据库中的坏表\n"
echo -e "mysql存储路径: mysql数据库文件存储的路径，路径配置在my.cnf中的字段datadir=/var/lib/mysql"
echo -e "mysql命令路径: mysql二进制路径，使用which mysql，可以查询\n"
pass=123456
name=root

read -p "输入mysql存储路径(默认为 datadir=/var/lib/mysql): "  choose
data_path=$choose
unset choose

read -p "请输入mysql命令路径(默认为 /usr/bin/mysql): " mysql_version
#标准输入、标准输出、标准错误输出的文件标示符 由 0、1、2标识
read -p "请选择是检查服务器上所有数据库还是指定的数据库 1:检查全部数据库 2:只检查指定数据库: " choose
if [ $choose == 1 ]; then
  cd $data_path
  for directory_list in $(ls)
    do
      if [ -d $directory_list ];then
          if [ "mysql" != "${directory_list}" -a "test" != "${directory_list}" ];then
              cd ${directory_list}
              echo "当前检查数据库为:"${directory_list}
              for file_list in $(ls *.frm)
              do
                _file_name=${file_list%.frm}
                echo -e "\n" >> /tmp/check_table_all.log
                ${mysql_version} -h 127.0.0.1 -u${name} -p${pass} -e "check table "${directory_list}.${_file_name} 2>&1 >> /tmp/check_table_all.log
              done
              cd ..
          fi
      fi
  done
      cat /tmp/check_table_all.log | grep "Table is marked as crashed" > /tmp/check_table_repair.log
      repair_count_all=` awk 'END{print NR}' /tmp/check_table_repair.log `
      echo -e "所有数据库用有${repair_count_all}张表需要修复！"

      ##repair
      awk -v bin="$mysql_version" -v mysqlname="$name" -v mysqlpw="$pass" '{print bin " -h127.0.0.1 -u"mysqlname" -p"mysqlpw" -e \"repair table " $1 "\""}' /tmp/check_table_repair.log|sh #-x
      more  /tmp/check_table_repair.log

      ##rm
      rm /tmp/check_table_all.log
      rm /tmp/check_table_repair.log
else
  read -p "请输入要检查的数据库名称: " db_name
  cd ${data_path}/${db_name}
  for file_list in $(ls *.frm)
    do
      _file_name=${file_list%.frm}
      echo -e "\n" >> /tmp/check_${db_name}.log
      ${mysql_version} -h 127.0.0.1 -u${name} -p${pass} -e "check table "${db_name}.$_file_name 2>&1 >> /tmp/check_${db_name}.log
    done
    cat /tmp/check_${db_name}.log | grep "Table is marked as crashed" > /tmp/check_${db_name}_repair.log
    repair_count=`awk 'END{print NR}' /tmp/check_${db_name}_repair.log`
    echo -e "${db_name}中共有${repair_count}个表需要修复！\n "

    ##repair
    awk -v bin="$mysql_version" -v mysqlname="$name" -v mysqlpw="$pass" '{print bin " -h127.0.0.1 -u"mysqlname" -p"mysqlpw" -e \"repair table " $1 "\""}' /tmp/check_${db_name}_repair.log|sh #-x
    more /tmp/check_${db_name}_repair.log

    ##rm
    rm /tmp/check_${db_name}.log
    rm /tmp/check_${db_name}_repair.log
fi