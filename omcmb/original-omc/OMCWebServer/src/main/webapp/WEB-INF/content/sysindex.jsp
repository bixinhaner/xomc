<%@ page contentType="text/html;charset=UTF-8"
	import="com.baicells.omc.busi.system.login.entity.UserInfo"%>
<%@page import="com.baicells.omc.busi.utils.ComConstants"%>
<%@page import="com.busi.mts.utils.MtsComConstants"%>
<%
	response.setHeader("Pragma", "No-cache");
	response.setHeader("Cache-Control", "no-cache");
	response.setDateHeader("Expires", 0);
	UserInfo ui = (UserInfo) session
			.getAttribute(ComConstants.SESSION_KEY);
%>
<%@ include file="/common/taglibs.jsp"%>
<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd">

<html xmlns="http://www.w3.org/1999/xhtml">
	<head>
		<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
		<title>${menuPath}--<fmt:message key="sys_title" bundle="${mts_titles}"/>：</title>
		
	<link rel="stylesheet" type="text/css" href="${ctx}/js/jquery-easyui/themes/${easyui_themes}/easyui.css" />
	<link rel="stylesheet" type="text/css" href="${ctx}/js/jquery-easyui/themes/icon.css" />
	<link rel="stylesheet" type="text/css" href="${ctx}/login/css/frame.css" />	
	<link rel="stylesheet" type="text/css" href="${ctx}/login/css/normal.css" />	
	<link rel="stylesheet" type="text/css" href="${ctx}/login/css/jquery.jscrollpane.css"  />
	<link href="${ctx}/css/admin.css" type="text/css" rel="stylesheet">

	<script type="text/javascript" src="${ctx}/js/My97DatePicker/WdatePicker.js"></script>
	<script type="text/javascript" src="${ctx}/js/jquery-easyui/jquery.min.js"></script>
	<script type="text/javascript" src="${ctx}/js/jquery-easyui/jquery.easyui.min.js"></script>
    <script type="text/javascript" src="${ctx}/js/utils.js"></script>
<style>
.admin_left .tree-node {
	padding:7px 0;
	height: 18px;
	white-space: nowrap;
	cursor: pointer;
	border-bottom:1px solid #d1d1d1;
}
.easyui-accordion{
	margin:0 8px 0 23px; /**/
	}
</style>

	</head>

	<body>
		<div id="layout_sys" class="easyui-layout" fit="true">
		
		
			<div data-options="region:'north'" border="0"
				style="height: 60px; overflow: hidden;">
				<div id="admin_header">
					<div class="hd_nav">
						<span class="welcome"><em><%=ui.getUsercode()%></em>欢迎您！</span>
						<span class="times"><a href="javascript:void(0);"  onclick="javascript:passwdUpd()" >修改密码</a></span>
						<span class="help"><a href="">帮助</a>
						</span>
						<span class="logout"><a href="javascript:void(0);"
							onclick="javascript:logout('${ctx}')">退出</a>
						</span>
								
											
					</div>
				</div>
			</div>
			
			<div id="west" data-options="region:'west'" border="0"
				style="width: 219px; overflow-y: hidden;overflow-x:hidden;">				
				<div class="admin_left" style="">
				
					<div class="easyui-accordion" style="height:360px;overflow-y: auto;">
						<a title="收起" style="cursor:pointer;color:red;font-weight:bold;" href="javascript:void(0);" onclick="javascript:sh()"> << </a>
						<ul id="admintree" class="easyui-tree" data-options="animate:true">

						</ul>
					</div>
				</div>
				<div class="admin_left_foot">
				</div>
			</div>
			
			
			<div id="center"  data-options="region:'center',border:false" border="0">
				<div class="easyui-layout" fit="true">
					<div data-options="region:'north'" border="0" style="height:34px;overflow:hidden;">
					
						<div class="admin_body">
							
							<div class="admin_location">							
							当前位置：${menuPath}<span id="locationPath"></span></div>
						</div>
					</div>
					<div id="mainpage" data-options="region:'center',border:false" border="0">
					</div>
				</div>	
			</div>


			<div data-options="region:'south'" border="0" style="height: 35px;">
				<div class="admin_footer">
					版权所有：<fmt:message key="copyright" bundle="${mts_titles}"/>
				</div>
			</div>
		</div>
<!-- <div id="passwd" class="easyui-window" title="密码修改"
	data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:600,height:400">
</div> -->



<div id="winopenFull" class="easyui-window" title="全景视图"
	data-options="modal:true,closed:true,draggable:false,minimizable:false,maximizable:false,onClose:function(){ 
		//fnTaskQuery();
	}" fit='true' />

</body>
</html>

<script>
	var site_url = '${ctx}';
	var __AG_SPT_CHAR = '<%=com.busi.mts.utils.StringPool.AG_SPT_CHAR%>';////亚通字符串分割符
	var __YES = '<%=com.baicells.omc.busi.utils.ComConstants.YES%>';
	var __NO = '<%=com.baicells.omc.busi.utils.ComConstants.NO%>';
	$().ready(function() {
		var pid = '${pid}';			
		var tree_url = site_url + "/system/sysGlb/getMenuTree.action?top_id="+pid;
		menutreeLoad('admintree', 'mainpage', '${ctx}', tree_url);
	});


	function passwdUpd() {

		/* $("#passwd").window("open");
		$("#passwd").window("refresh","${ctx}/login/goModifyPassword.action"); */
		openDefaultWindow("${ctx}/login/goModifyPassword.action",{
    		title: '密码修改',
    		width:600,height:400
    	});
	}
		

	function sh() {
		
		$('#layout_sys').layout('collapse','west');
	}

</script>


<!-- tips start -->
<!-- Tooltip classes -->
<style>
.tag_grp {
	z-index: 99999, display :   inline-block;
	color: black;
	float: none;
	font-size: 12px;
	padding: 2px 5px;
	margin: 2px;
	cursor: pointer;
	border: #d3d3d3 1px solid;
	line-height: 22px;
	border-radius: 3px;
}
</style>

<link rel="stylesheet"
	href="${ctx}/js/plus/poshytip/tip-yellow/tip-yellow.css"
	type="text/css" />



<script type="text/javascript"
	src="${ctx}/js/plus/poshytip/jquery.poshytip.js"></script>
<!-- tips end -->