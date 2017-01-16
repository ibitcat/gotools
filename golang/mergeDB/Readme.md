golang 实现的合服工具。

实现思路：
1. 首先清理每个数据库的无用数据（比如超过30天没用进入过游戏的玩家）；
   将查找的死号玩家id插入到一张缓存表temp中，然后根据该缓存表清理除player表以外的其他数据表，
   接着清理有依赖关系的表（比如：好友、公会等），最后清理player表
2. 合并数据库
   检查两个数据库的表数量是否一致，然后检查主数据库中的表在从数据库（被吃掉的）是否存在，
   再检查每个要合并的表，表结构是否一致，最后尝试合并，合并成功后，清除从数据表的数据。

配置：

```json
{
	"worker":2,//工作携程数量
	"master_db":"domi_temp",//主数据库名
	"slave_db":[//从数据库
		"domi_temp2"
	],
	"exclude":[// 排除不需要合并的表
		"commonCd"
	],
	"special":[// 需要特殊处理的表
		{
			"name":"guild",
			"sql":["update guild_member,(select pid,office from (select * from guild_member ORDER BY office ASC) as a GROUP BY a.gid) as b set guild_member.office=1 WHERE b.office>1 and b.pid=guild_member.pid;"]
		},
		{
			"name":"guild_member",
			"sql":["delete from guild where id not in (SELECT gid from guild_member GROUP BY gid)"]
		},
		{
			"name":"friend",
			"sql":["DELETE friend FROM friend,temp_for_merge WHERE friend.pid=temp_for_merge.pid or friend.fpid=temp_for_merge.pid"]
		},
		{
			"name":"friend_blacklist",
			"sql":["DELETE friend_blacklist FROM friend_blacklist,temp_for_merge WHERE friend_blacklist.blackPid=temp_for_merge.pid"]
		},
		{
			"name":"warTeam",
			"sql":[
				"update warTeam set leader = 0 where leader in (select pid from temp_for_merge)",
				"update warTeam set member1 = 0 where member1 in (select pid from temp_for_merge)",
				"update warTeam set member2 = 0 where member2 in (select pid from temp_for_merge)",
				"delete from warTeam where leader = 0 and member1= 0 and member2=0",
				"update warTeam set leader = (CASE WHEN member1 <>0 THEN member1 ELSE member2 END) where leader = 0",
				"update warTeam set member1 = 0 where leader = member1",
				"update warTeam set member2 = 0 where leader = member2"
			]
		}
	]
}
```
