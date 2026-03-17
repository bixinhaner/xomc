<%@ page contentType="text/html;charset=UTF-8"%>
<%@page import="com.baicells.omc.busi.utils.ComConstants"%>
<%@page import="com.baicells.omc.busi.system.login.entity.UserInfo"%>
<%
	response.setHeader("Pragma","No-cache");
	response.setHeader("Cache-Control","no-cache");
	response.setDateHeader("Expires", 0);

	UserInfo ui = (UserInfo)session.getAttribute(ComConstants.SESSION_KEY);
%>
<%@ include file="/common/taglibs.jsp"%>
<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd">
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
<link type="text/css" href="${ctx}/css/common.css" rel="stylesheet" />
<script type="text/javascript" src="${ctx}/js/jquery-easyui/jquery.min.js"></script>
<title></title>
</head>
<body>
<div id="header">
	<div class="logo"><img src="${ctx}/images/logo.png" alt="<fmt:message key="sys_title" bundle="${mts_titles}"/>" title="<fmt:message key="sys_title" bundle="${mts_titles}"/>" /></div>
    <div class="hd_nav">
    	<span class="welcome"><em><%=ui.getUsercode() %></em>欢迎您！</span>
        <span class="times">上次登录的时间是2013-10-25</span>
        <span class="help"><a href="">帮助</a></span>
        <span class="logout"><a href="javascript:void(0);"  onclick="javascript:logout('${ctx}')" >退出</a></span>
    </div>
    <div class="banner"><img src="${ctx}/images/banner2.gif" /></div>
</div>
<div id="container">
	<ul class="menu">
		${menuHtml}
		<!-- 
		<li><a  target="_blank" href="${ctx}/m/navi.action?toPage=main" class="menu1">CPA</a></li>	
    	<li><a  target="_blank" href="${ctx}/login/toIdx.action?fwd=rpt" class="menu2">审单结果查询</a></li>
        <li><a  target="_blank" href="${ctx}/login/toIdx.action?fwd=mts" class="menu3">人工干预</a></li>
        <li class="last"><a  target="_blank" href="${ctx}/dq/dqnavi.action?toPage=main" class="menu4">BI审单分析</a></li>
     
        <li><a  target="_blank" href="${ctx}/dq/dqnavi.action?toPage=main" class="menu5">DQ</a></li>
        <li><a  target="_blank" href="" class="menu6">管理驾驶舱</a></li>
        <li><a  target="_blank" href="${ctx}/login/toIdx.action?fwd=moni" class="menu7">监控</a></li>
        <li class="last"><a  target="_blank" href="${ctx}/login/toIdx.action?fwd=adm" class="menu8">系统管理</a></li>
         -->
    </ul>
</div>
<div id="footer">版权所有：<fmt:message key="copyright" bundle="${mts_titles}"/></div>
</body>
</html>
<script type="text/javascript">
	var systest = 22;
	function logout(ctx) {
		var params = {};
		$.ajax({
			url: ctx+'/login/logout.htm',
		    type: 'POST',
		    dataType: 'text',
		    data :params,
		    ontentType: "application/x-www-form-urlencoded; charset=utf-8",
		    error: function(request){
			    window.location.href = ctx+"/";
		    },
		    success: function(request){
		    }
		});
	}
</script>