<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>

<style type="text/css">
.divDivideLine{
	max-width:800px;
	width:750px;
	margin: 30px 90px;
	border: 1px solid #DCECF7;
	clear : both;
}
.eNBExportBlock{
	margin:10px 30px 10px 60px;
}
.timeRange{
	display:inline-block;
}
.exportContent{
	margin:10px 0 20px 30px;
}
.exportDatagrid{
	width:370px;
	height:470px;
	border:1px solid #DCECF7;
	display:inline-block;
	margin-right:10px;
}
.eNBSelectBlock{
	width:370px;
	height:500px;
	display:inline-block;
	margin-right:10px;
}
.titleSpanStyle{
	margin: 7px 0;
	display:block;
}
#selectedENBDiv .datagrid-body .datagrid-cell{
	position:relative;
	cursor:pointer;
}
#selectedENBDiv .datagrid-view2 .datagrid-body td{
	cursor:pointer;
}
#selectedENBDiv td{
	border:none;
}
.upgrade_delete{
	visibility:hidden;
	transation: visibility 0.5s ease 0.2s;
	cursor:pointer;
	position:absolute;
	top:0px;
	width:20px;
	height:20px;
}
.visible .upgrade_delete {
	visibility:visible;
}
.el-card__body{
	display:flex;
	flex-direction:column;
	flex:1 1 auto;
	height:100%;
	overflow:auto;
}
</style>

<!-- 窗口，新建自定义查询模板-->
<div style='display:flex;flex-direction:column;height:100%;'>
	<div class="el-card__header">
		<span><%=rb.getString("DaoChu")%></span>
		<span class='el-icon el-icon-close' onclick="closeExportENBCount()"></span>
	</div>
	<div class='el-card__body'>
		<div style='width:100%;background:#fff;'>
			<div class="eNBExportBlock">
				<div class="group-title not-extend">
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("Title_KPI_ShiJianFanWei")%></span>
				</div>
				<div class="exportContent">
					<div class="timeRange">
			            <span class="titleSpanStyle"><%=rb.getString("KaiShiShiJian")%><%=rb.getString("MaoHao")%></span>
			            <input id="export_start_time" class="easyui-datetimebox border border-box" data-options="editable:false" style="height: 26px;width:200px;">
			        </div>
			        <span style="margin:0 31px;">--</span>
			        <div class="timeRange">
			            <span class="titleSpanStyle"><%=rb.getString("JieShuShiJian")%><%=rb.getString("MaoHao")%></span>
			            <input id="export_end_time" class="easyui-datetimebox border border-box" data-options="editable:false" style="height: 26px;width:200px;">
			        </div>
				</div>
			</div>
			<div class="divDivideLine"></div>
			<div class="eNBExportBlock">
				<div class="group-title not-extend">
					<span class="title-icon"></span>
					<span class="title-text">eNB <%=rb.getString("XuanZe")%></span>
				</div> 
				<div class="exportContent">
					<div class="eNBSelectBlock" style='margin-right:5px;float:left'>
						<span class="titleSpanStyle">eNB <%=rb.getString("XuanZe")%>:</span>
						<div class="exportDatagrid">
							<table id="eNB_export_datagrid" class="panelTableDiv">
						</table></div>
					</div>
					<div class="eNBSelectBlock" id="selectedENBDiv" style='margin-right:0px;'>
						<span class="titleSpanStyle"><%=rb.getString("YiXuan")%> eNB:</span>
						<div class="exportDatagrid">
							<table id="eNB_export_selected" class="panelTableDiv"></table>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
	<div class="el-card__footer">
		<a href="#" class="linkbutton" style='margin-top:10px;' onclick="exportENBCount()"><span><%=rb.getString("DaoChu")%></span></a>
	</div>
	
</div>
<!-- 用户工具栏 -->
<div id="toolbar_eNB_export_datagrid" class="toolbarContainer">
    <div class="queryGroup">
        <input id="eNB_export_datagrid_Input" name="search_text" style="margin-left:0px;width:260px;" placeholder="ECI" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
        <b class="el-icon el-icon-common-search" onclick='$("#eNB_export_datagrid").datagrid("reload")'></b>
    </div>
</div>
<%-- 上传文件的用的表单 --%>
<form enctype="multipart/form-data" method="post" style="display:none" id="enbCountExportForm">
    <input name="startTime"  value="">
    <input name="endTime" value="">
    <input name="cellIds" value="">
    <input name="timeZone" value="">
</form>
<script type="text/javascript">
var deleteArr=[];
$(function(){
    closeLoading();
    var export_start_time = formatDate(new Date(gloableTime)).substring(0,10)+' 00:00:00';
    var export_end_time = formatDate(new Date(gloableTime));
    setTimeout(function(){
    	$('#export_start_time').datetimebox("setValue",export_start_time);
        $('#export_end_time').datetimebox("setValue",export_end_time);
    },800);
    // 表格数据
    $("#eNB_export_datagrid").datagrid({
    	url : '${ctx}/egw/monitor/queryEnbMonitorPageList.action',
		border : false,
		fit : true,
		fitColumns : true,
        rownumbers : true,
		striped : true,
		singleSelect : false,
		idField : 'cell_id',
        toolbar:'#toolbar_eNB_export_datagrid',
		pagination : true,
		pagePosition : 'bottom',
        onBeforeLoad : beforeLoad_eNB_export_datagrid,
		onLoadSuccess : loadSuc_eNB_export_datagrid,
		onSelect: onSelectedENBeGW,
        onUnselect: onUnselectedENBeGW,
        onSelectAll: onSelectedAllENBeGW,
        onUnselectAll: onUnselectedAllENBeGW,
	    columns : [[
	    			{field: 'ck',checkbox:true, width: 100},
	    			{field: 'cell_id',sortable:true, width: 80, title: 'ECI' },
	    			{field: 'enb_id',sortable:true,width: 100, title: '<%=rb.getString("EnodebId")%>'},
	    			{field: 'ip', sortable:true,width: 100, title: 'IP'},
	    		]],
	});
    
    $('#eNB_export_selected').datagrid({
        border:false,
        rownumbers : true,
        striped : true,
        singleSelect: true,
        fit:true,
        fitColumns:true,
        onLoadSuccess : loadSuc_eNB_export_selected,
        data:[],
  	    columns : [[ 
					{field: 'value', hidden: true},
					{field: 'text',width:300,title:titleClear,formatter:deleteSelectRow}
		          ]]
  	 });
    $("#eNB_export_datagrid_Input").bind("keyup", function(e){
		if (e.keyCode == 13){
			$("#eNB_export_datagrid").datagrid("reload");
			$("#eNB_export_datagrid_Input").blur();
		}	
	});
})
//加载前事件-基站列表
function beforeLoad_eNB_export_datagrid(param){
	param["gwIp"] = gwIp;
	param["gwPort"] = gwPort;
	param["timeZone"] = timeZone;
	var search_text = $("#eNB_export_datagrid_Input").val();
	if(search_text != ""){
		param["searchText"] = search_text;
	}
}
function loadSuc_eNB_export_datagrid(data) {
    $(this).datagrid("enableContextmenuAutoSize");
    $(this).datagrid("fixRownumber");
}
/**
* 基站列表列表选择事件
* @param index{object}   下标
* @param row{object}   行数据
*/
function onSelectedENBeGW(index,row) {
	var grid = $("#eNB_export_selected");
	var cwidth = grid.datagrid('getColumnOption','text').width;
	var pwidth = $("#selectedENBDiv .datagrid-view2").width();
	var row = {
			value:row["cell_id"],
			text:row["cell_id"]
	}
	grid.datagrid("appendRow", row);
	var cData = grid.datagrid("getRows");
	if(pwidth - cwidth >10){
		var cl = $("#eNB_export_selected").datagrid('getColumnOption','text');
		cl.width = pwidth;
		$("#eNB_export_selected").datagrid({fitColumns:true,data:cData});
	}else{
		$("#eNB_export_selected").datagrid({fitColumns:false,data:cData});
	}
}
/**
* 取消选择事件
* @param index{object}   下标
* @param row{object}   行数据
*/
function onUnselectedENBeGW(index,row){
	var cell_id = row["cell_id"];
	var allRows = $("#eNB_export_selected").datagrid("getRows");
	allRows.map(function(item,index){
		if(item.value ==cell_id ){
			$("#eNB_export_selected").datagrid("deleteRow",index);
		}
	})
}
// 全选
function onSelectedAllENBeGW(){
	var selCell = $("#eNB_export_datagrid").datagrid("getSelections");
	var del = $("#eNB_export_selected").datagrid("getRows");
	var selArr = [];
	del.map(function(item,index){
		selArr.push(item.value);
	})
	selCell.map(function(item,index){
		if(selArr.indexOf(item["cell_id"])!=-1){
			return;
		}else{
			var row = {
					value:item["cell_id"],
					text:item["cell_id"]
			}
			$("#eNB_export_selected").datagrid("appendRow", row);
		}
	})
}
// 取消全选
function onUnselectedAllENBeGW(){
	$("#eNB_export_selected").datagrid("loadData", {total:0,rows:[]});
}
// 数据加载成功
function loadSuc_eNB_export_selected(data) {
    $("#selectedENBDiv .datagrid-view2 .datagrid-body").bind("mouseover",function(){
		var td = event.target
		if(event.target.tagName=='SPAN'||event.target.tagName=='TD'||(event.target.tagName=='DIV'&& $(event.target).hasClass('datagrid-cell'))){
			var td = event.target;
			if(event.target.tagName=='TD'){
				
			}else{
				td = $(event.target).parents('td')[0];
			}
			td.classList.add('visible');
			var grid = $("#eNB_export_selected");
			var cwidth = grid.datagrid('getColumnOption','text').width;
			var pwidth = $("#selectedENBDiv .datagrid-view2").width();
			$($(td).find("span")).css("left",pwidth-50);
		}
	}).bind("mouseout",function(){
		event.target.classList.remove('visible');
	});
	$("#selectedENBDiv").bind("mouseout",function(){
		$('#selectedENBDiv .visible').removeClass('visible');
	});
}
// 导出页面关闭
function closeExportENBCount(){
	$("#eNBCountDetailPanel").animate({right:'-2000px'},500);
}
// 导出
function exportENBCount(){
	var cellIds = [];
	var selectedENBData= $("#eNB_export_selected").datagrid("getRows");
	if(selectedENBData.length>0){
		$.each(selectedENBData,function(index,ele){
			cellIds.push(ele.value)
		})
		var startTime = $("#export_start_time").datetimebox("getValue");
		var endTime = $("#export_end_time").datetimebox("getValue");
		cellIds =cellIds.join(",") ;
		
		$("#enbCountExportForm [name=startTime]").val(startTime);
	    $("#enbCountExportForm [name=endTime]").val(endTime);
	    $("#enbCountExportForm [name=cellIds]").val(cellIds);
	    $("#enbCountExportForm [name=timeZone]").val(timeZone);
	    /* $("#enbCountExportForm").form('submit', {
	        url: "${ctx}/egw/monitor/downloadMonitorCellIdInfos.action",
	        success: function (data) {
	        	closeExportENBCount();
	        },
	        error:function (data){
	        	if(typeof data == 'string'){
	        		var data = JSON.parse(data);
	        	}
	        	showMsg('error_msg',data["message"]);
	        },
	        onSubmit: function(param){
				var bool = checkParams(param)
				if(!bool) return false;
	        }
	    }); */
	    exportByForm("${ctx}/egw/monitor/downloadMonitorCellIdInfos.action",{
	    	startTime: startTime,
	    	endTime: endTime,
	    	cellIds: cellIds,
	    	timeZone: timeZone
	    });
	}else{
		showMsg('prompt_msg','<%=rb.getString("QingXuanZeSheBei")%>');
	}
}
// 已选ECI 表格标题
function titleClear(){
	var span = $('<p onclick="clearAllSelect()" style="position:absolute;top:0px;left:75%;line-height:38px;cursor:pointer;"><span class="el-icon el-icon-operation-delete" style="font-size:16px;margin-top:10px;margin-right:5px;"></span><span style="float:right;color:#4D84FF;margin-right:10px;"><%=rb.getString("QingKong")%></span><span class="upgrade_clear"></span></p>');
	var title = $(this);
	title.after(span);
	return "<%=rb.getString("YiXuan")%> ECI"
}
/**
*  表格数据 清空 格式化数据
* @param value{string}   绑定值
* @param rowData{object}   行数据
* @param rowIndex{number}   下标
*/ 
function deleteSelectRow(value,rowData,rowIndex){
	var delStr = "<span onclick='deleteRow(\""+rowData.value+"\")' class='upgrade_delete'></span>"
	return value+delStr;
}
// 清空已选
function clearAllSelect(){
	$("#eNB_export_selected").datagrid("loadData", {total:0,rows:[]});
	$("#eNB_export_datagrid").datagrid("clearSelections");
}
// 移除单个
function deleteRow(value){
	var grid = $.data($("#eNB_export_datagrid")[0],'datagrid');
	grid.selectedRows = grid.selectedRows.filter(function(item){
		return item["cell_id"]!=value;
	})
	grid.checkedRows = grid.checkedRows.filter(function(item){
		return item["cell_id"]!=value;
	})
	var rows = $("#eNB_export_selected").datagrid("getRows");
	rows.map(function(item,index){
		if(value == item.value){
			$("#eNB_export_selected").datagrid("deleteRow",index);
		}
	})
	var allRows = $("#eNB_export_datagrid").datagrid("getRows");
	allRows.map(function(item,index){
		if(value == item["cell_id"]){
			 $("#eNB_export_datagrid").datagrid("unselectRow",index);
		}
	})
}
</script>