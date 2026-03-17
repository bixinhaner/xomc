/* 格式化时间 */
function dateformatter(date) {
	var y = date.getFullYear();
	var m = date.getMonth() + 1;
	var d = date.getDate();
	var h = date.getHours();
	var min = date.getMinutes();
	var s = date.getSeconds();
	return y + '-' + (m < 10 ? ('0' + m) : m) + '-' + (d < 10 ? ('0' + d) : d)
			+ " " + (h < 10 ? ('0' + h) : h) + ":"
			+ (min < 10 ? ('0' + min) : min) + ":" + (s < 10 ? ('0' + s) : s);
}

function dateParser(s) {
	if (!s) {
		return new Date();
	}
	var reg = /(\d+)-(\d+)-(\d+) (\d+):(\d+):(\d+)/;
	var arr = reg.exec(s);
	return new Date(arr[1], arr[2] - 1, arr[3], arr[4], arr[5], arr[6]);
}

// 获得当前指定时区的时间
function getNowZoneTime(time_zone) {
	//var system_time_zone = -1 * new Date(gloableTime).getTimezoneOffset();

	//var nowDate = new Date(new Date(gloableTime).getTime() + 1000 * 60 * (time_zone - system_time_zone));
	var nowDate = new Date(gloableTime);
	var nowTime = dateformatter(nowDate);
	return nowTime;
}

function getNowDateAndTime(timeLevel, n) {
	var now = new Date(gloableTime);
	if (n) {
		if (day < n) {
			if (month > 1) {
				month -= month;
			} else {
				year -= 1;
				month = 12;
			}
		}
		now.setDate(now.getDate() - n);
	}
	var year = now.getFullYear();
	var month = now.getMonth() + 1;
	var day = now.getDate();
	var hh = now.getHours();
	var mm = now.getMinutes();
	var ss = now.getSeconds();

	if (timeLevel == 1) {
		mm = parseInt(mm / 15) * 15;
	}
	var clock = year + "-";

	if (month < 10)
		clock += "0";
	clock += month + "-";

	if (day < 10)
		clock += "0";
	clock += day + " ";

	if (hh < 10)
		clock += "0";
	clock += hh + ":";

	if (mm < 10)
		clock += "0";
	clock += mm + ":";

	if (ss < 10)
		clock += "0";
	clock += ss;

	return clock;
}

// 时间处理 获得当前时间指定时区的时间范围 hours参数为开始时间偏移小时数
function getNowTimeToZoneTimeRange(time_zone, hours) {
	//var system_time_zone = -1 * new Date(gloableTime).getTimezoneOffset();
	var nowDate = new Date(gloableTime);
	var dataStartDate = new Date(nowDate.getTime() - 1000 * 60 *60 * hours);
	var dataStartTime = dateformatter(dataStartDate);

	//var dataEndDate = dateformatter(nowDate);
	//new Date(nowDate.getTime() + 1000 * 60* (time_zone - system_time_zone));
	var dataEndTime = dateformatter(nowDate);
	var params = {
		time_zone : time_zone,
		start_time : dataStartTime,
		end_time : dataEndTime
	};
	return params;
}

// 验证开始时间和结束时间是否合法
function validateStartAndStopTime(startTimeStr, endTimeStr) {
	if (isNotNull(startTimeStr) && isNotNull(endTimeStr)) {
		var startDate = dateParser(startTimeStr);
		var endDate = dateParser(endTimeStr);
		if (startDate.getTime() < endDate.getTime()) {
			return "true";
		}
	}
	if (!isNotNull(startTimeStr) && isNotNull(endTimeStr)) {
		return "true";
	}
	if (isNotNull(startTimeStr) && !isNotNull(endTimeStr)) {
		return "true";
	}
	if ("" == startTimeStr && "" == endTimeStr) {
		return "true";
	}
	return "false";
}

function isNotNull(arg) {
	if (arg == null) {
		return false;
	}
	if (arg == "") {
		return false;
	} 
	return true;
}

// 日期偏差 - 分钟
function addTimes(date, times) {
	var d = new Date(date);
	d = d.valueOf();
	d = d + times * 60 * 1000;
	a = new Date(d);
	return a;
}

// 日期偏移 - 天
/*function addDate(date, num) {
	var d = new Date(date);
	d = d.valueOf();
	d = d + num * 24 * 60 * 60 * 1000;
	a = new Date(d);
	return a;
}*/
function addDate(date, days){
  let newDate = new Date(date);
  newDate.setDate(newDate.getDate() + days); 
  return newDate;
}

// 时间差
function differ(pre, after) {
	var preDate = new Date(pre), afterDate = new Date(after);
	var differTimes = preDate.valueOf() - afterDate.valueOf();
	return Math.round(differTimes / (24 * 60 * 60 * 1000));
}
