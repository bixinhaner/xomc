<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<%-- 窗口-日志-确认收集 --%>
<div id="winConfirmPeriodCollect" style="width: 100%;height: 100%">
	 <div class="easyui-layout" data-options="border:false, fit:true">
	 	<div region="north" data-options="border:false,height:500" style="padding: 15px 20px;">
	    	<table id="confirmPeriodCollectLogDg" class="easyui-datagrid"
	 			data-options="fit:true,
							singleSelect: false,
							striped: true,
							border:true,
							pagination:false, 
							idField:'small_cell_code',
							title:'<%=rb.getString("JiZhanLieBiao")%>',
							rownumbers:true,
							fitColumns:true,onLoadSuccess:datagridLoadSuccess">
				<thead>
					<tr>
						<th data-options="field:'small_cell_code',hidden:true"></th>
						<th data-options="field:'serial_number'" width="450"><%=rb.getString("XiaoZhanBianMa")%></th>
						<th data-options="field:'host_name'" width="450"><%=rb.getString("HostName")%></th>
					</tr>
				</thead>
	 		</table>
	 	</div>
	 	<div region="center" data-options="border:false">
			<div class="easyui-panel" data-options="fit:true,border:false" style="padding: 0px 20px">
				<div class="easyui-layout" data-options="fit:true,border:false">
					<div region="center" data-options="border:true,title:'<%=rb.getString("ShangBaoZhouQi")%>'">
						<div style="margin-top: 20px;margin-left: 20px;">
							<label style="width: 90px; display: inline-block" class="borderBoxClass"><%=rb.getString("FenZhongZhouQi")%><%=rb.getString("MaoHao")%></label>
							<select id="periodReportLogMin" class="border border-box borderBoxClass">
								<option value="900" selected="selected">15</option>
								<option value="1800">30</option>
								<option value="2700">45</option>
								<option value="3600">60</option>
							</select>
						</div>
						<div style="margin-top: 20px;margin-left: 20px;">
							<label style="width: 90px; display: inline-block" class="borderBoxClass"> <%=rb.getString("KaiShiShiJian")%><%=rb.getString("MaoHao")%></label>
							<input id="periodReportLogStartTime" class="easyui-datetimebox border border-box borderBoxClass" style="vertical-align: middle; height: 26px">
							<label style="width: 90px; display: inline-block; margin-left: 10px" class="borderBoxClass"><%=rb.getString("JieShuShiJian")%><%=rb.getString("MaoHao")%></label>
							<input id="periodReportLogEndTime" class="easyui-datetimebox border border-box borderBoxClass" style="vertical-align: middle; height: 26px">
						</div>
					</div>
				</div>
		   </div>
	 	</div>
	 	<div region="south" data-options="border:false,height:46" style="padding: 10px 20px 10px 0px;">
	    	<a onclick="cancelPeriodReportLogFile();" style="float:right;" class="easyui-linkbutton"><%=rb.getString("QuXiao")%></a>
	    	<a onclick="confirmPeriodReportLogFile();" style="float:right;margin-right:15px;" class="easyui-linkbutton"><%=rb.getString("QueDing")%></a>
	 	</div>
	 </div>
</div>