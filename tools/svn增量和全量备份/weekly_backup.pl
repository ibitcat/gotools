#!/usr/bin/perl -w 
my $svn_repos="/var/svn/repos/proj-h5"; 
my $backup_dir="/var/svn/backup"; 
my $next_backup_file = "weekly_fully_backup.".`date +%Y%m%d`; 

$youngest=`svnlook youngest $svn_repos`; 
chomp $youngest; 

print "Backing up to revision $youngest\n"; 
my $svnadmin_cmd="svnadmin dump --revision 0:$youngest $svn_repos >$backup_dir/$next_backup_file"; 
`$svnadmin_cmd`; 
open(LOG,">$backup_dir/last_backed_up"); #记录备份的版本号 
print LOG $youngest; 
close LOG; 
#如果想节约空间，则再执行下面的压缩脚本 
print "Compressing dump file...\n"; 
print `gzip $backup_dir/$next_backup_file`;

