#!/usr/bin/perl -w 
my $svn_repos="/var/svn/repos/proj-h5"; 
my $backup_dir="/var/svn/backup"; 
my $next_backup_file = "daily_incremental_backup.".`date +%Y%m%d`; 

open(IN,"$backup_dir/last_backed_up"); 
$previous_youngest = <IN>; 
chomp $previous_youngest; 
close IN; 

$youngest=`svnlook youngest $svn_repos`; 
chomp $youngest; 
if ($youngest eq $previous_youngest) 
{ 
  print "No new revisions to backup.\n"; 
  exit 0; 
} 
my $first_rev = $previous_youngest + 1; 
print "Backing up revisions $youngest ...\n"; 
my $svnadmin_cmd = "svnadmin dump --incremental --revision $first_rev:$youngest $svn_repos > $backup_dir/$next_backup_file"; 
print `$svnadmin_cmd`; 

#记录备份的版本号
open(LOG,">$backup_dir/last_backed_up"); 
print LOG $youngest; 
close LOG;

#如果想节约空间，则再执行下面的压缩脚本 
print "Compressing dump file...\n"; 
print `gzip $backup_dir/$next_backup_file`; 

