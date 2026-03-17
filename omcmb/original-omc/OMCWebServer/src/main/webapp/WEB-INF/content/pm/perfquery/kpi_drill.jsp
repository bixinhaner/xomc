<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>

<style type="text/css">
	#kpiDrillDatagrid_header{
		position: absolute;
		width: 100%;
		line-height: 50px;
		height: 50px;
		top: 0px;
		color: #7993B6;
		font-size: 16px;
	}
	
	#kpiDrillDatagrid_body{
		position: absolute;
		width: 96%;
		top: 60px;
		left: 20px;
		right: 20px;
		bottom: 10px;
		overflow: auto;
	}
</style>

<!--查看统计指标详情  -->
<div id="kpiDrillDatagrid_header">
	<div id="kpiDrillTitle" style="display:inline-block;margin-left:30px; color: rgba(0, 0, 0, 0.8);"><%=rb.getString("ZhiBiaoXiangQing")%></div>
	<div class="circleIcon" style="right:15px;">
		<span class="el-icon el-icon-circle-close" onclick="closeKpiDrillDiv()" style='font-size: 12px !important; line-height: 26px; float: unset !important; margin: 0 !important;position: unset !important;'></span>
	</div>
</div>
<div id="kpiDrillDatagrid_body" class="tabsContentDiv">
	<div style="height:100%;width:100%" >
		<table id="kpiDrillDatagrid"></table>
	</div>   		
</div>

<script type="text/javascript">

	$(function(){
		// 表格数据加载
		$("#kpiDrillDatagrid").datagrid({
			border:false,
			fit:true,
			fitColumns:true,
			singleSelect:true,
			rownumbers:true,
			striped:true,
			singleSelect:true,
			idField:'startTime',
			onLoadSuccess: datagridLoadSuccess,
			columns: [[
				{field: 'formula',width: 100, title: '<%=rb.getString("JiSuanGongShi")%>(<%=rb.getString("ZhiBiaoID")%>)'},
				{field: 'formulaName',width: 100, title: '<%=rb.getString("JiSuanGongShi")%>(<%=rb.getString("ZhiBiaoMingCheng")%>)'},
				{field: 'timeLevel',width: 70,title: '<%=rb.getString("ShiJianLiDu")%>(<%=rb.getString("FenZhongDaXie")%>)'},
				{field: 'startTime',width: 60, title: '<%=rb.getString("KaiShiShiJian")%>'},
				{field: 'counterDetail',width: 130, title: '<%=rb.getString("JieGuo")%>'},
			]]
		})	
	    closeLoading();
	})
</script>