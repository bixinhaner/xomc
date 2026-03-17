<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo ui = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>

<style>
	.rightHideDiv{
		position:absolute;
		left:0px;
		right:0px;
		bottom:0px;
		top:0px;
		background:#FFFFFF;
		z-index:100;
		display:none;
		}
</style>
<div id='egwConfigLoading'
	style="display:none;position: absolute; z-index: 1000; top: 0px; left: 0px;right:0px;bottom:0px;background: #FFFFFF; background-image: url(${ctx}/css/images/global/loading.gif); background-position: center center;background-repeat: no-repeat;">
</div>
<!-- eGW config界面 -->
<div class="panelDefault" style='overflow:hidden'>
	<div id='eGWConfigDiv' class="singleContentDiv" style="top:0;">		
		<table id="eGWconfigTable" class="panelTableDiv"></table>
	</div>
</div>

<!-- eGW设置 工具栏 -->
<div id="toolbar_eGWConfig" class="toolbarContainer">
	<div class="queryGroup">
		<input id="eGWConfigSearch" name="task_name" placeholder="<%=rb.getString("EGWMingCheng")%> / <%=rb.getString("EGWIP")%>" />
		<b class='el-icon el-icon-common-search' onclick="queryeGWGconfigTable()"></b>
	</div>
</div>
<!-- 右侧滑出  edit页面   -->
<div id="editConfigDiv" class="rightHideDiv"></div>
<script>
var paramsegwIp = "";
var paramsegwPort = "";
$(function() {
	
    closeLoading();
	
    $("#eGWConfigSearch").bind("keyup", function(e){
		if (e.keyCode == 13){
			queryeGWGconfigTable();
		}
	}); 
    var configdata ={
	    	'total':100,
	    	'row':[
	    	       {
	    	    	   gw_id:1,
	    	    	   gw_ip:'192.168.5.4',
	    	    	   gw_name:'gateway1',
	    	    	   gw_port:'2802'
	    	       },
	    	       {
	    	    	   gw_id:2,
	    	    	   gw_ip:'192.168.5.3',
	    	    	   gw_name:'gateway2',
	    	    	   gw_port:'2801'
	    	       }
	    	 ]
	    };
    $("#eGWconfigTable").datagrid({
    	 url:'${ctx}/egw/register/queryEgwPageList.action',
    	/* data:configdata.row, */ 
    	border: false,
    	fit: true,
    	queryParams : {
            timeZone : timeZone
        },
    	rownumbers:true,
    	fitColumns: true,
        striped: true,
        singleSelect: true, 
        idField: 'gw_id',
        toolbar:'#toolbar_eGWConfig',
        pagination: true,
        pagePosition: 'bottom',
        columns: [[
    		{field:'gw_id', hidden:true},
    		{field:'gw_name',sortable:true,fixed:false,width:350,title:'<%=rb.getString("EGWMingCheng")%>'},
    		{field:'gw_ip',sortable:true,width:350,fixed:false,title:'<%=rb.getString("EGWIP")%>'},
    		{field:'gw_port',sortable:true,fixed:false,width:350, title: '<%=rb.getString("EGWDuanKou")%>'},
    		{field:'egwConfig_operation',width:140,fixed:true,formatter:eGWConfigFormatter,title:'<%=rb.getString("CaoZuo")%>'},
    		
    	]],
        onBeforeLoad:beforeload_eGWConfig,
        onLoadSuccess:function(){
        	$(this).datagrid("fixRownumber");
        	$(this).datagrid("enableContextmenuAutoSize");
        }
    });
    
}) 
	//假数据   假装是从后台请求过来的数据
	var editConfigData;  /* = {
			gwip:'192.168.5.4',
			gwport:'2802',
			success:"true",
			plmn:"3443",
			uplinkWSlAPToEgwIp:"192.189.1.1",
			uplinkWSlAPToMMEIp:"192.189.1.2",
			s1MmeIpToEnb:["192.168.9.1","192.168.9.2","192.168.9.3","192.168.9.4"],
			s1MmePortToEnb:"36678",
			egwUplinkGtpuIp:["192.178.56.1"],
			egwDownlinkGtpuIp:["192.168.9.1","192.168.9.2","192.168.9.3"],
			eNB_Config:[
			  {enb_id:32,tac:1},
			{enb_id:33,tac:1},
			{enb_id:34,tac:1}
			],
			s1FlexEnable:"1",
			logLevel:"4"

	} */ 
	
	
//加载前事件-eGW 设置界面
function beforeload_eGWConfig(param) {
	param["searchText"] = $("#eGWConfigSearch").val();
	param["timeZone"] = timeZone;
	
}
//操作格式化
function eGWConfigFormatter(value, rowData, rowIndex){
	var rowDates = rowData;
	var gwIp = rowDates.gw_ip;
	var gwPort = rowDates.gw_port;
	value = "<div class='el-icon el-icon-operation-edit CODE_EGW hidden' style='margin-left:15px;' title='modify' onclick='modifyeGWConfig(\""+gwIp+"\",\""+gwPort+"\")'></div>"
	return value;
}
//eGW Config 修改操作
var linkData = {};
function modifyeGWConfig(eGWIp,eGWPort){
	paramsegwIp = eGWIp;
	paramsegwPort = eGWPort;
	$('#egwConfigLoading').show();
	var params = {};
	params["gwIp"] = eGWIp;
	params["gwPort"] = eGWPort;
	linkData["gwIp"] = eGWIp;
	linkData["gwPort"] = eGWPort;
	var url = '${ctx}/egw/config/toConfigEdit.action';
	$.post("${ctx}/egw/config/getBasicConfig.action",params,function(data){
		editConfigData = data; 
		if(data['success']){
			$('#egwConfigLoading').hide();
			$("#editConfigDiv").slideDown(300);
			$("#editConfigDiv").panel({
				href:url,
				fit:true
				});
			//加载edit config
		}else{
			$('#egwConfigLoading').hide();
			showMsg('error_msg',data["message"]);
		}
	},"json"); 
	
}
function queryeGWGconfigTable(){
	$("#eGWconfigTable").datagrid('reload');
}
</script>