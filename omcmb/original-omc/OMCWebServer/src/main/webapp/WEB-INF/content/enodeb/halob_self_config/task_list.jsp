<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>

<div class="panelDefault" style="overflow:hidden;">
	<div class="singleContentDiv">
		<table id="taskManagement_selfConfig"></table>
	</div>
	<%-- 任务管理 查看  bottom:-800px; --%>
	<div class="slideDiv" style="height:400px;border:1px solid rgb(74,179,255);overflow-y:auto;bottom:-800px;">
		<div class="slideHeader" style="border:none;">
			<h3><%=rb.getString("ZhiXingQingKuang")%></h3>
			<ul class="iconText">
				<li><a class="titleIcon_close iconSize" onclick="closeSlideDiv()"></a></li>
			</ul>
		</div>
		<div style="padding:0 20px;position:absolute;top:61px;bottom:1px;left:0px;right:0;">
			<table id="taskManagement_selfConfig_view"></table>
		</div>
	</div>
</div>

<!-- 工具栏-操作日志  -->
<div id="toolbar_taskManagement_selfConfig" class="admin_query_head toolbarContainer" style="position:relative">
	   	<div style="display:inline-block;margin-right:20px;">
		   <label style="font-size:13px;margin-left:5px;margin-top:-3px;vertical-align:top;"><%=rb.getString("ZiKaiZhanZhiXingFangShi")%><%=rb.getString("MaoHao")%></label>
           <select id="chose_carryOut_type" class="inputDivCss border border-box" style="padding-top:0px;height:26px;width:200px;">
               <option value=""><%=rb.getString("QuanBu")%></option>
               <option value="0"><%=rb.getString("ZiDongZhiXing")%></option>
               <option value="1"><%=rb.getString("ShouDongZhiXing")%></option>
           </select>
	    </div>
        <div class="queryGroup">
       	   <input id="searchSelfConfigDevice" name="search_text" style="" placeholder="<%=rb.getString("XiaoZhanBianMa")%>">
	   	   <b class="searchResultImgChangeStyle" onclick="queryTaskManage_selfConfig()"></b>
        </div>
</div>    
<script type="text/javascript">
$(function() {
	closeLoading();
	//任务管理 表格数据加载
 	$("#taskManagement_selfConfig").datagrid({
   		border:false,
   		fit:true,
       	url:'${ctx}/cell/halobSelfConfig/getAllAutoConfigTaskList.action',
       	queryParams : {
       		timeZone : timeZone,
			like_fields : 'serial_number'
		},
       	toolbar:'#toolbar_taskManagement_selfConfig',
       	singleSelect:true,
       	rownumbers:true,
       	fitColumns:true,
       	pagination:true,
       	pagePosition:'bottom',
       	striped:true,
       	singleSelect:true,
       	idField:'task_id',
       	onBeforeLoad : taskManagement_selfConfig,
       	onLoadSuccess:datagridLoadSuccess,
       	columns: [[
                    {field: 'task_id', hidden: true},
                    {field: 'connection_status',width: 20,formatter: connStatusFormatterSelfconfig},
                    {field: 'serial_number',sortable:true,width: 100, title: '<%=rb.getString("XiaoZhanBianMa")%>'},
                    {field: 'execute_type',width: 100, title: '<%=rb.getString("ZiKaiZhanZhiXingFangShi")%>',formatter: executeType_selfCfg},
                    {field: 'execute_procedure',width: 100, title: '<%=rb.getString("JinDu")%>',formatter: executeProcedure_selfCfg},
                    {field: 'status',width: 100, title: '<%=rb.getString("ZhuangTai")%>',formatter: status_selfCfg},
                    {field: 'result',width: 100, title: '<%=rb.getString("JieGuo")%>',formatter: result_selfCfg},
                    {field: 'start_time',sortable:true,width: 100, title: '<%=rb.getString("KaiShiShiJian")%>'},
                    {field: 'operate',fixed:true,width: 135,title: '<%=rb.getString("CaoZuo")%>',formatter: operate_taskManagement_selfConfig},
                    
         	]],
    	})
	//详细信息查看 表格数据加载
 	$("#taskManagement_selfConfig_view").datagrid({
   		border:false,
   		fit:true,
       	singleSelect:true,
       	rownumbers:true,
       	fitColumns:true,
       	pagination:true,
       	pagePosition:'bottom',
       	striped:true,
       	singleSelect:true,
       	idField:'id',
       	onLoadSuccess:datagridLoadSuccess,
       	columns: [[
                    {field: 'id', hidden: true},
                    {field: 'progress',width: 80, title: '<%=rb.getString("JinDu")%>',formatter: view_progress_selfConfig},
                    {field: 'status',width: 140,title: '<%=rb.getString("ZhuangTai")%>',formatter: view_status_selfCfg},
                    {field: 'start_time',sortable:true,width: 70, title: '<%=rb.getString("KaiShiShiJian")%>'},
                    {field: 'end_time',sortable:true,width: 70, title: '<%=rb.getString("JieShuShiJian")%>'},
                    
         	]],
    	})
    	$("#searchSelfConfigDevice").bind("keyup", function (event) {
            if (event.keyCode == 13) {
            	queryTaskManage_selfConfig();
            }
        });
	$("#chose_carryOut_type").change(function(){
		queryTaskManage_selfConfig();
	})
})
// 基站连接状态-格式化
function connStatusFormatterSelfconfig(value, rowData, rowIndex) {
	var synTime  = rowData.lastsyntime;
	if ("On" == value) {
		return "<div class='conn_on'><div>";
	} else if("updating" == value){
		return "<div class='status_syncing'></div>";
	}else if ("Exception" == value) {
		return "<div class='conn_exc'><div>";
	}else {
		if(rowData["have_connected"] == "2"){
			return "";
		}
		return "<div class='conn_off'><div>";
	}
	return value;
}
function taskManagement_selfConfig(param){
	var search_text = $("#searchSelfConfigDevice").val();
    if (search_text != "") {
        param["search_text"] = search_text;
    }
    param["execute_type"] = $("#chose_carryOut_type").val();
    
}

var ChaKan = '<%=rb.getString("ChaKan")%>';
var ShanChu = '<%=rb.getString("ShanChu")%>';
var XiaYiBu = '<%=rb.getString("XiaYiBu")%>';
var ChongxinZhiXing = '<%=rb.getString("ChongxinZhiXing")%>';

function operate_taskManagement_selfConfig(value, rowData, rowIndex){
	var gridData = JSON.stringify(rowData);
    value = "";
    value = value + "<div class='operationDiv operation_view' title='"+ChaKan+"' onclick='viewTaskMana(" + gridData + ")'></div>";
    if(rowData.execute_type == "1"){
	    if(rowData.status == "1" || rowData.execute_procedure == "5"){
		    value = value + "<div class='operationDiv operation_next_disabled eNbSelfConfiguration hidden' id='"+rowData.task_id+"_"+rowIndex+"' title='"+XiaYiBu+"' style='margin-left:15px;'></div>";
	    }else{
		    value = value + "<div class='operationDiv operation_next eNbSelfConfiguration hidden' id='"+rowData.task_id+"_"+rowIndex+"' title='"+XiaYiBu+"' style='margin-left:15px;' onclick='nextTaskMana(" + gridData +","+rowIndex+ ")'></div>";
	    }
	    if(rowData.status != "2" ){
		    value = value + "<div class='operationDiv operation_refresh_disabled eNbSelfConfiguration hidden' title='"+ChongxinZhiXing+"' style='margin-left:15px;'></div>";
	    }else{
		    value = value + "<div class='operationDiv operation_refresh eNbSelfConfiguration hidden' title='"+ChongxinZhiXing+"' style='margin-left:15px;' onclick='rerunTaskMana(" + gridData + ")'></div>";
	    }
    }
    value = value + "<div class='operationDiv operation_delete eNbSelfConfiguration hidden' title='"+ShanChu+"' style='margin-left:15px;' onclick='deleteTaskMana(" + gridData + ")'></div>";
    return value;
}
//查看 
function viewTaskMana(gridData){
	$(".slideDiv").animate({bottom:'0px'},500);
	var task_id = gridData.task_id;
    $("#taskManagement_selfConfig_view").datagrid({
		url:'${ctx}/cell/halobSelfConfig/getCurrAutoConfigTaskRecord.action',
	   	queryParams : {
	   		timeZone : timeZone,
	   		task_id: task_id
		}
    })
}
//下一步
function nextTaskMana(gridData){
	var task_id = gridData.task_id;
	$.post("${ctx}/cell/halobSelfConfig/doAutoConfigNextStep.action", {task_id:task_id}, function (data) {
    	if (data["success"]) {
    		//$("#taskManagement_selfConfig").datagrid("reload");
        	var index = $("#taskManagement_selfConfig").datagrid("getRowIndex",task_id);
        	$("#"+task_id+"_"+index).removeClass('operation_next').addClass('operation_next_disabled');
        	$("#taskManagement_selfConfig_view").datagrid({
        		url:'${ctx}/cell/halobSelfConfig/getCurrAutoConfigTaskRecord.action',
        	   	queryParams : {
        	   		timeZone : timeZone,
        	   		task_id: task_id
        		}
            });
    	}else{
    		$.messager.alert(TiShi, data["message"]);
                return;
    	}
    }, "json");
}
//重新执行
function rerunTaskMana(gridData){
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenChongXinZhiXing")%>", function (r) {
        if (r) {
        	var task_id = gridData.task_id;
            $.post("${ctx}/cell/halobSelfConfig/goRerunAutoConfig.action", {task_id:task_id}, function (data) {
            	if (data["success"]) {
	            	$("#taskManagement_selfConfig").datagrid("reload");
	            	$("#taskManagement_selfConfig_view").datagrid("reload");
            	}else{
            		$.messager.alert(TiShi, data["message"]);
	                    return;
            	}
            }, "json");
        }
    }).addClass("seriousConfirm");
}
//删除
function deleteTaskMana(gridData){
	closeSlideDiv();
	var task_id = gridData.task_id;
	$.post("${ctx}/cell/halobSelfConfig/isAutoConfigTaskEnd.action", {task_id:task_id}, function (data) {
    	if (data["success"]) {
    		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenZiPeiZhiWanChengShanChuCiRenWu")%>", function (r) {
    	        if (r) {
		    		$.post("${ctx}/cell/halobSelfConfig/delAutoConfigTask.action",  {task_id:task_id}, function (data) {
		            	if (data["success"]) {
			            	$("#taskManagement_selfConfig").datagrid("reload");
			            	$("#taskManagement_selfConfig_view").datagrid("reload");
		            	}else{
		            		$.messager.alert(TiShi, data["message"]);
			                    return;
		            	}
		    		}, "json");
    	        }	
        	}).addClass("seriousConfirm");
    	}else{
    		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenZiPeiZhiWanChengShanChuCiRenWu")%>", function (r) {
    	        if (r) {
    	            $.post("${ctx}/cell/halobSelfConfig/delAutoConfigTask.action",  {task_id:task_id}, function (data) {
    	            	if (data["success"]) {
    		            	$("#taskManagement_selfConfig").datagrid("reload");
    		            	$("#taskManagement_selfConfig_view").datagrid("reload");
    	            	}else{
    	            		$.messager.alert(TiShi, data["message"]);
    		                    return;
    	            	}
    	            }, "json");
    	        }
    	    }).addClass("seriousConfirm");
    	}
    }, "json");
	
}
function closeSlideDiv(){
	$(".slideDiv").animate({bottom:'-800px'},500);
}
function executeType_selfCfg(value, rowData, rowIndex){
	if(rowData.execute_type == "0"){
		value = "<%=rb.getString("ZiDongZhiXing")%>";
	}else{
		value = "<%=rb.getString("ShouDongZhiXing")%>";
	}
	return value;
}
function executeProcedure_selfCfg(value, rowData, rowIndex){
	if(rowData.execute_procedure == "1"){
		value = "<%=rb.getString("RuanJianShengJi")%>";
	}else if(rowData.execute_procedure == "2"){
		value = "<%=rb.getString("LicenseXiaFa")%>";
	}else if(rowData.execute_procedure == "3"){
		value = "<%=rb.getString("CanShuPeiZhi")%>";
	}else if(rowData.execute_procedure == "4"){
		value = "<%=rb.getString("WeiKaiShi")%>";
	}else if(rowData.execute_procedure == "5"){
		value = "<%=rb.getString("JieShu")%>";
	}
	return value;
}
function status_selfCfg(value, rowData, rowIndex){
	if(rowData.status == "0"){
		value = "<%=rb.getString("WeiKaiShi")%>";
	}else if(rowData.status == "1"){
		value = "<%=rb.getString("JinXingZhong")%>";
	}else if(rowData.status == "2"){
		value = "<%=rb.getString("WanCheng")%>";
	}
	return value;
}
function result_selfCfg(value, rowData, rowIndex){
	if(rowData.result == "1"){
		value = "<%=rb.getString("ChengGong")%>";
	}else if(rowData.result == "0"){
		value = "<%=rb.getString("ShiBai")%>";
	}else{
		value = "";
	}
	return value;
}
function view_progress_selfConfig(value, rowData, rowIndex){
	if(rowData.progress == "1"){
		value = "<%=rb.getString("RuanJianShengJi")%>";
	}else if(rowData.progress == "2"){
		value = "<%=rb.getString("LicenseXiaFa")%>";
	}else if(rowData.progress == "3"){
		value = "<%=rb.getString("CanShuPeiZhi")%>";
	}
	return value;
}
function view_status_selfCfg(value, rowData, rowIndex){
	value = value.replace('JinXingZhong',"<%=rb.getString("JinXingZhong")%>")
	.replace('ChengGong',"<%=rb.getString("ChengGong")%>")
	.replace('ShiBai',"<%=rb.getString("ShiBai")%>")
	.replace('DouHao',"<%=rb.getString("DouHao")%>")
	.replace('MaoHao',"<%=rb.getString("MaoHao")%>")
	.replace('LicenseGuanLiMeiYouDuiYingSN',"<%=rb.getString("LicenseGuanLiMeiYouDuiYingSN")%>")
	.replace('ShengJiChaoShi',"<%=rb.getString("ShengJiChaoShi")%>")
	.replace('WenJianBuCunZai',"<%=rb.getString("WenJianBuCunZai")%>")
	.replace('ZiPeiZhiZhongMeiYouDuiYingWenJian',"<%=rb.getString("ZiPeiZhiZhongMeiYouDuiYingWenJian")%>")
	.replace('RuanJianShengJiGuiHuaMeiYouDuiYingBanBen',"<%=rb.getString("RuanJianShengJiGuiHuaMeiYouDuiYingBanBen")%>")
	.replace('CellIdChongFuQingChongXinShuRu',"<%=rb.getString("CellIdChongFuQingChongXinShuRu")%>")
	
	return value;
}
function queryTaskManage_selfConfig(){
	$("#taskManagement_selfConfig").datagrid('reload');
}
</script>