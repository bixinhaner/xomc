<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo ui = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>

<style>
    .formatterDiv{
    	vertical-align:middle;
    	display:inline-block;
    	width:17px;
    	height:17px;
    	cursor:pointer;
    }
	.rightHideDiv{
		position:absolute;
		left:15px;
		right:15px;
		bottom:15px;
		top:10px;
		background:#FFFFFF;
		z-index:100;
		display:none;
	}
</style>
<!-- eGW 重启界面 -->
<div class="panelDefault" style='overflow:hidden'>
	<div class="singleTitle">
		<span><%=rb.getString("EGWChongQi")%></span>
	</div>
	<div id='eGWRebootDiv' class="singleContentDiv">		
		<table id="eGWRebootTable" class="panelTableDiv"></table>
	</div>
</div>
<!-- eGW重启 工具栏 -->
<div id="toolbar_eGWReboot" class="toolbarContainer">
	<div class="queryGroup">
		<input id="eGWRebootSearch" name="task_name" placeholder="<%=rb.getString("EGWMingCheng")%> / <%=rb.getString("EGWIP")%>" />
		<b class="el-icon el-icon-common-search" onclick="queryeGWRebootTable()"></b>
	</div>
</div>
<script>
$(function(){
	closeLoading;
	 //键盘回车事件  --- 基站监控界面高级查询
	$("#eGWRebootSearch").bind("keyup", function(e){
		if (e.keyCode == 13){
			queryeGWRebootTable()
		}
	}); 
	$("#eGWRebootTable").datagrid({
		url:'${ctx}/egw/reboot/queryRebootEgwPageList.action',
		border:false,
		fit:true,
		queryParams:{
			timeZone:timeZone,
		},
		rownumbers:true,
		fitColumns:true,
		striped:true,
		singleSelect:true,
		idField:'gw_id',
		toolbar:'#toolbar_eGWReboot',
		pagination:true,
		pagePosition:'bottom',
		columns:[[
		       {field:'gw_id',hidden:true},
		       {field:'gw_name',sortable:true,width:200,title:'<%=rb.getString("EGWMingCheng")%>'},
		       {field:'gw_ip',sortable:true,width:100,fixed:false,title:'<%=rb.getString("EGWIP")%>'},
	    	   {field:'gw_port',sortable:true,fixed:false,width:100, title: '<%=rb.getString("EGWDuanKou")%>'},
	    	   {field:'reboot_time',sortable:true,fixed:false,width:200, title: '<%=rb.getString("ChongQiShiJian")%>'},
	    	   {field:'task_progress',sortable:true,fixed:false,width:90,formatter:eGWRebootProgressFormatter,title: '<%=rb.getString("JinDu")%>'},
	    	   {field:'task_result',sortable:true,fixed:false,width:350, title: '<%=rb.getString("JieGuo")%>'},
	    	   {field:'egwerboot_operation',width:70,fixed:true,formatter:eGWRebootFormatter,title:'<%=rb.getString("CaoZuo")%>'},
		]],
		onBeforeLoad:beforeLoad_eGWReboot
	})
})
//加载前事前 -- eGW重启界面
function beforeLoad_eGWReboot(param){
	param['searchText'] = $("#eGWRebootSearch").val();
	param['timeZone'] = timeZone;
}
/**
* 重启格式化
* @param value{string} 绑定值
* @param rowData{object}  行数据
* @param rowIndex{number}  下标
*/
function eGWRebootFormatter(value,rowData,rowIndex){
	var gwId = rowData.gw_id;
	var gwIp = rowData.gw_ip;
	var gwPort = rowData.gw_port;
	var taskProgress = rowData.task_progress;
	if(taskProgress == '0'){
		var value = "<div class='el-icon el-icon-operation-reboot disabled' style='cursor:not-allowed;width:100%;text-align:center;' title='reboot'></div>"
	}else{
		var value = "<div class='el-icon el-icon-operation-reboot CODE_EGW hidden' style='width:100%;text-align:center;' title='reboot' onclick='rebooteGW(\""+gwId+"\",\""+gwIp+"\",\""+gwPort+"\")'></div>"
	}
	
	return value; 
}
/**
* 进度格式化
* @param value{string} 绑定值
* @param rowData{object}  行数据
* @param rowIndex{number}  下标
*/
function eGWRebootProgressFormatter(value,rowData,rowIndex){
	if(value == '0'){
		return "<%=rb.getString("JinXingZhong")%>";
	}else if(value == '1'){
		return "<%=rb.getString("YiJieShu")%>";
	}
	
}
//查询表格
function queryeGWRebootTable(){
	$("#eGWRebootTable").datagrid('reload');
}
/**
* 重启操作
* @param gwId{string} id
* @param gwIp{string}  IP
* @param gwPort{string}  端口
*/
function rebooteGW(gwId,gwIp,gwPort){
	var params = {};
	params['gwId'] = gwId;
	params['gwIp'] = gwIp;
	params['gwPort'] = gwPort;
	$.messager.confirm({
		title:QueRen,
		msg:'<%=rb.getString("QueDingChongQiWangGuan")%>',
		fn:function(r){
			if(r){
				$.post('${ctx}/egw/reboot/rebootEgw.action',params,function(data){
					if(data["success"]){
						$("#eGWRebootTable").datagrid('reload');
					}else{
						$("#eGWRebootTable").datagrid('reload');
						
					}
				},'json')
			}
		}
	}).addClass("seriousConfirm");
}
</script>