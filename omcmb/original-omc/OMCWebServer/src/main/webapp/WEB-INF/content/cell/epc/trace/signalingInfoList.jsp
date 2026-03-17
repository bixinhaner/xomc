<%@ page import="java.util.Locale" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>
<style>

</style>
	<div id="toolbar_view_trace_export" class="toolbar_view_container">
		<span class="panelTitle"><%=rb.getString("JieGuo")%></span>
		<span class="panelLittleTitle"></span>
		
		<span class="el-icon el-icon-close" title='<%=rb.getString("GuanBi")%>' style="padding-left:20px;margin-right:30px;float:right;font-size:20px;" onclick='closeCheckTrace()'></span>
		<span class="el-icon el-icon-operation-export" title='<%=rb.getString("DaoChu")%>' style="padding-left:0px;margin-right:0px;float:right;font-size:20px;" onclick='exportSignaling()'></span>
		<span id="biggestBtn" class="biggestBtn" title='<%=rb.getString("ZuiDaHua")%>' style="" onclick='viewBiggest()'></span>
	</div>
	<table id="tableTraceViewList"></table>
	
<%-- 导出信令信息用的表单 --%>
<form method="post" style="display: none"  id="signalingDownloadFileForm"></form>
<div id="winTest" class="easyui-window" title="<%=rb.getString("XinLingXiaoXi")%>"
     data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:true,width:1050,height:600,resizable:true,inline:false,draggable:true">
</div>

<script type="text/javascript">
var traceInfiID = "${traceId}";
iSInpprogresstraceId = traceInfiID;
$(function(){
	$(".panelLittleTitle").text('('+'<%=rb.getString("GenZongCanKaoHao")%>'+':'+traceInfiID+')');
	$("#tableTraceViewList").datagrid({
		url:'${ctx}/signaling/querySignalingInfoList.action',
		//data:traceViewdata,  
		border:false,
		fit:true,
		queryParams:{
			timeZone:timeZone,
		},
		rownumbers:true,
		fitColumns:true,
		striped:true,
		singleSelect:true,
		idField:'id',
		toolbar:'#toolbar_view_trace_export',
	    pagination:true,
		pagePosition:'bottom', 
	    queryParams:{
			traceId:traceInfiID,
			timeZone:timeZone
		},   
		columns:[[
		       {field:'id',fixed:false,width:120,hidden:true, title: 'id'},
		       {field:'traceId',fixed:false,width:120,hidden:true, title: '<%=rb.getString("GenZongCanKaoHao")%>'},
		       {field:'file_name',fixed:false,width:120,hidden:true, title: '<%=rb.getString("WenJianMing")%>'},
		       {field:'interface',fixed:false,width:120, title: '<%=rb.getString("JieKouLeiXing")%>'},
	    	   {field:'direction',fixed:false,width:120, title: '<%=rb.getString("FangXiang")%>'},
	    	   {field:'protocol',fixed:false,width:150, title: '<%=rb.getString("XieYi")%>'},
	    	   {field:'message',fixed:false,formatter:openTraceFoematter,width:100, title: '<%=rb.getString("XiaoXi")%>'},
	    	   {field:'report_time_sec',fixed:false,width:100,sortable:true,formatter:timensFormatter,title: '<%=rb.getString("ShiJian")%>'},
	    	   {field:'report_time_ns',fixed:false,hidden:true,width:100, title: 'weimiao'},
		]],
	   /* onBeforeLoad:beforeLoad_SASMonitor, */
	   onLoadSuccess:loadSuccess_tableTraceViewList
	})
})
//导出信令追踪信息
function exportSignaling(){
	var params = {
            timeZone: timeZone
        };
	params.traceId = traceInfiID;
	$.post("${ctx}/signaling/createSignalingPcapFile.action",params,function(data){
		if(data["success"]){
			/* $("#signalingDownloadFileForm").form('submit', {
		        url: "${ctx}/signaling/exportSignalingPcapFile.action",
		        queryParams:params,
		        onSubmit: function(param){ 
		        }
		    }); */
			exportByForm("${ctx}/signaling/exportSignalingPcapFile.action",params);
		}else{
			$.messager.alert(TiShi,data["message"]);
		}
	},"json");
	
}
//message格式化函数
function openTraceFoematter(value,rowData,rowIndex){
	return "<a class='' style='color:#1DA3FC;cursor:pointer;' onclick='openTrace(\""+rowData.id+"\")'>" + value + "</a>";
}
function timensFormatter(value,rowData,rowIndex){
	return "<span>"+ value+"."+ rowData.report_time_ns+"</span>";
}
//查看详细信令   二进制  树相互转换
function openTrace(id){
	var params = {
			id:id
	}
	$.post("${ctx}/signaling/isExistSignalingRelateFile.action", params, function(data){
        if (data["success"]) {
        	$("#winTest").window("open").window('center'); 
        	$("#winTest").window("refresh", "${ctx}/signaling/toSignalingDetailInfo.action?id="+id); 
        } else {
        	showMsg('prompt_msg','<%=rb.getString("XinLinXiaoXiBuCunZai")%>');
        }
    }, "json");
	
}
//查看面板最大化
function viewBiggest(){
	if($("#biggestBtn").hasClass("biggestBtn")){
		$(".viewPanelContainer").animate({"bottom":"2px","height":"100%"},function(){
			$("#biggestBtn").removeClass("biggestBtn").addClass("restoreBtn");
			$("#tableTraceViewList").datagrid("resize",{
				height:($(".viewPanelContainer").height())
			});
		});
	}else{
		$(".viewPanelContainer").animate({"bottom":"2px","height":"288px"},function(){
			$("#biggestBtn").removeClass("restoreBtn").addClass("biggestBtn");
			$("#tableTraceViewList").datagrid("resize",{
				height:($(".viewPanelContainer").height())
			});
		});
		
	}
	
}
function loadSuccess_tableTraceViewList(){
	$(this).datagrid("fixRownumber");
	$(this).datagrid("enableContextmenuAutoSize");
}
</script>
