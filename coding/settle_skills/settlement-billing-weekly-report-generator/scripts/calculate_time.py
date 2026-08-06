import datetime

def get_duty_cycle():
    # 本周的值班周期（上周四 00:00 至 下周三 23:59，基于执行时间点所在的周期）
    now = datetime.datetime.now()
    # weekday: 0=Mon, 1=Tue, 2=Wed, 3=Thu, 4=Fri, 5=Sat, 6=Sun
    weekday = now.weekday()
    
    # 找到最近的周四
    if weekday >= 3: # 周四及以后
        start_date = now - datetime.timedelta(days=(weekday - 3))
    else: # 周一到周三
        start_date = now - datetime.timedelta(days=(weekday + 4))
        
    start_date = start_date.replace(hour=0, minute=0, second=0, microsecond=0)
    end_date = start_date + datetime.timedelta(days=6)
    end_date = end_date.replace(hour=23, minute=59, second=59, microsecond=999999)
    
    return start_date, end_date

if __name__ == "__main__":
    start, end = get_duty_cycle()
    print(f"START_DATE={start.strftime('%Y-%m-%d')}")
    print(f"END_DATE={end.strftime('%Y-%m-%d')}")
    print(f"START_TIME={start.strftime('%Y-%m-%dT%H:%M:%S+08:00')}")
    print(f"END_TIME={end.strftime('%Y-%m-%dT%H:%M:%S+08:00')}")
