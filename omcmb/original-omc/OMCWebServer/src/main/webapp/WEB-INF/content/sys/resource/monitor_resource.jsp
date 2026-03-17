<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>

<!-- 系统资源管理界面 -->
<div id="resourceLayout" style="width: 100%;height: 100%;min-height: 700px;">
	<div id="northLayout" style="background:#F5F7FA;height:29%;margin-bottom:15px;">
		<div class="splitPanel" id="northLayout" style="height:100%;width:50%;float:left;">
			<div class="singleTitle">
				<%=rb.getString("ChuLiQi")%>
			</div>
			<div class="contentDiv"  style="padding-top:20px;">
				<table class="easyui-datagrid" fit="true" fitColumns="true"
					data-options="border:false,singleSelect:true,rownumbers:true,striped:true,onLoadSuccess:datagridLoadSuccess,
						url: '${ctx}/system/resourceMonitor/getCPUInfo.action'">
                    <thead>
	                    <tr>
	                        <th data-options="field:'cpuUserUsage'" width="80"><%=rb.getString("YongHuShiYong")%>(%)</th>
	                        <th data-options="field:'cpuSystemUsage'" width="80"><%=rb.getString("XiTongShiYong")%>(%)</th>
	                        <th data-options="field:'cpuTotalUsagePt'" width="80"><%=rb.getString("ZongGongShiYong")%>(%)</th>
	                        <th data-options="field:'cpuFreePercent'" width="80"><%=rb.getString("ShengYu")%>(%)</th>
	                    </tr>
                    </thead>
               	</table>
			</div>
		</div>
		<div class="splitPanel" id="northLayout" style="height:100%;width:49%;float:right;">
			<div class="singleTitle">
				<%=rb.getString("CiPan")%>
			</div>
			<div class="contentDiv"  style="padding-top:20px;">
				<table class="easyui-datagrid"
					data-options="border:false,fitColumns:true,fit:true,singleSelect:true,rownumbers:true,onLoadSuccess:datagridLoadSuccess,
						url: '${ctx}/system/resourceMonitor/getFileSystemInfo.action',striped:true">
					<thead>
						<tr>
							<th data-options="field:'devName'" width="80"><%=rb.getString("CiPanHao")%></th>
					     	<th data-options="field:'sysTotalSize'" width="80"><%=rb.getString("ZongGongDaXiao")%>(MB)</th>
					     	<th data-options="field:'sysFreeSize'" width="80"><%=rb.getString("ShengYuDaXiao")%>(MB)</th>
					     	<th data-options="field:'sysUsedSize'" width="80"><%=rb.getString("ShiYongDaXiao")%>(MB)</th>
						</tr>
					</thead>
				</table>
			</div>
		</div>
	</div>
	<div class="splitPanel" id="northLayout" style="height:33%;margin-bottom:15px;">
		<div class="singleTitle">
			<%=rb.getString("NeiCun")%>
		</div>
		<div class="contentDiv"  style="padding-top:20px;">
			<table class="easyui-datagrid"
				data-options="border:false,fit:true,fitColumns:true,singleSelect:true,rownumbers:true,onLoadSuccess:datagridLoadSuccess,
					url: '${ctx}/system/resourceMonitor/getMemoryInfo.action',
					striped:true">
				<thead>
					<tr>
						<th data-options="field:'memoryTotal'" width="80"><%=rb.getString("ZongGongDaXiao")%>(MB)</th>
						<th data-options="field:'memoryFree'" width="80"><%=rb.getString("ShengYuDaXiao")%>(MB)</th>
						<th data-options="field:'memoryFreePt'" width="80"><%=rb.getString("ShengYu")%>(%)</th>
						<th data-options="field:'memoryUsed'" width="80"><%=rb.getString("ShiYongDaXiao")%>(MB)</th>
						<th data-options="field:'memoryUsedPt'" width="80"><%=rb.getString("ShiYongDe")%>(%)</th>
					</tr>
				</thead>
			</table>
		</div>
	</div>
	<div class="splitPanel" id="northLayout" style="height:33%;">
		<div class="singleTitle">
			<%=rb.getString("ShuJuKu")%>
		</div>
		<div class="contentDiv downTabs"  style="padding-top:20px;">
			<div class="easyui-tabs" data-options="fit:true,border:false">
				<div title="<%=rb.getString("DangQianShuJuKuXinXi")%>" style="padding: 20px;">
					<table class="easyui-datagrid" 
						data-options="border:false,fitColumns:true,fit:true,singleSelect:true,onLoadSuccess:datagridLoadSuccess,
							rownumbers:true,
							url:'${ctx}/system/resourceMonitor/getDatabaseInfo.action',
							striped:true">
						<thead>
							<tr>
								<th data-options="field:'threads_connected'" width="80"><%=rb.getString("LianJieXianChengShu")%></th>
								<th data-options="field:'threads_running'" width="80"><%=rb.getString("YunXingXianChengShu")%></th>
								<th data-options="field:'max_connections'" width="80"><%=rb.getString("ZuiDaXianChengShu")%></th>
								<th data-options="field:'data_size'" width="80"><%=rb.getString("ShuJuShiYongDaXiao")%>(MB)</th>
								<th data-options="field:'index_size'" width="80"><%=rb.getString("SuoYinShiYongDaXiao")%>(MB)</th>
								<th data-options="field:'free_size'" width="80"><%=rb.getString("ShengYuDaXiao")%>(MB)</th></tr>
						</thead>
					</table>
				</div>
				<div title="<%=rb.getString("LiShiShuJuKuXinXi")%>" style="padding: 20px 20px 0;">
					<table class="easyui-datagrid" fitColumns="true"
						data-options="border:false,fit:true,singleSelect:true,rownumbers:true,idField: 'time',onLoadSuccess:datagridLoadSuccess,
							pagination: true,url: '${ctx}/system/resourceMonitor/getDBSpaceRecord.action',
							striped:true">
						<thead>
						<tr>
							<th data-options="field:'time'" width="80"><%=rb.getString("RiQi")%></th>
							<th data-options="field:'data_size'" width="80"><%=rb.getString("ShuJuShiYongDaXiao")%>(MB)</th>
							<th data-options="field:'index_size'" width="80"><%=rb.getString("SuoYinShiYongDaXiao")%>(MB)</th>
							<th data-options="field:'free_size'" width="80"><%=rb.getString("ShengYuDaXiao")%>(MB)</th>
							<th data-options="field:'increase_size'" width="80"><%=rb.getString("ShiYongZengZhangDaXiao")%>(MB)</th>
						</tr>
						</thead>
					</table>
				</div>
			</div>
		</div>
	</div>
</div>

<script type="text/javascript">
function resizeResourcePanel(width, height) {
	$("#resourceLayout").layout("panel", "north").panel({height: height / 3});
	$("#resourceLayout").layout("panel", "south").panel({height: height / 3});
	$("#northLayout").layout("panel", "west").panel({width: width / 2});
}
</script>