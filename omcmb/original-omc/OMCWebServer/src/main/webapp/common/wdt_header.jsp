<%@ page contentType="text/html;charset=UTF-8" import="com.baicells.omc.busi.system.login.entity.UserInfo"%>
<%@page import="com.baicells.omc.busi.utils.ComConstants"%>
<%@page import="com.busi.mts.utils.MtsComConstants"%>
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
<link rel="stylesheet" type="text/css" href="${ctx}/js/jquery-easyui/themes/${easyui_themes}/easyui.css">
<link rel="stylesheet" type="text/css" href="${ctx}/js/jquery-easyui/themes/icon.css">
<link rel="stylesheet" type="text/css" href="${ctx}/css/admin.css">
<script type="text/javascript" src="${ctx}/js/jquery-easyui/jquery.min.js"></script>
<script type="text/javascript" src="${ctx}/js/jquery-easyui/jquery.easyui.min.js"></script>

<script type="text/javascript" src="${ctx}/js/utils.js"></script>



<title>sysTreeWidget</title>

<style>
.tag_grp{ display: inline-block;color: black;float: none;font-size: 12px;padding: 2px 5px;margin: 2px; cursor:pointer; border:#d3d3d3 1px solid; line-height:22px; border-radius:3px; }
</style>

</head>
