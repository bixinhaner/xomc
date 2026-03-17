<%@ page contentType="text/html; charset=utf-8"%>
<%@ taglib uri="http://java.sun.com/jsp/jstl/core" prefix="c"%>
<c:set var="ctx" value="${pageContext.request.contextPath}"/>
<%
	response.setStatus(HttpServletResponse.SC_OK);
%>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
<title>系统异常信息 </title>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
</head>
<body>
<style type="text/css">
<!--
body {
	margin: 20px;
	font-family: Roboto;
	font-size: 14px;
	color: #333333;
	background-color: #FFFFFF;
}

a {
	color: #1F4881;
	text-decoration: none;
}
-->
</style>
<script type="text/javascript">
	// 登录提交
	function loginpage() {
		window.location.href = '${ctx}';
	}
</script>
<div
	style="border: #cccccc solid 1px; padding: 20px; width: 500px; margin: auto"
	align="center">系统异常，请重试，如果重复出现异常请联系客服人员! <br />
<br />
<a href='#' onclick="return loginpage();">【返回首页】</a></div>
<br/>
<hr width="80%">
<h2><font color=#DB1260>LForum Error Page</font></h2>

<p>发生异常: <b>
</pre>
<hr width=80%>
<div style="border: 0px; padding: 0px; width: 500px; margin: auto">
<strong>版权所有:</strong> (C)&nbsp;LForum For Java&nbsp;2008</div>
</body>
</html>