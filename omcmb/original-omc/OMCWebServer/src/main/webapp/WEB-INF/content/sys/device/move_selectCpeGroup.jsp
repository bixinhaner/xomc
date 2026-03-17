<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<%-- 窗口-选择移动到的设备组(CPE) --%>
<div id="winSelectCpeGroupToMove" style="width: 100%;height: 100%;">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="padding: 10px;">
			<table class="easyui-datagrid" id="gridSelectCpeGroupToMove"
					data-options="fit:true,rownumbers:true,fitColumns:true,border:false,striped:true,singleSelect:true,onLoadSuccess:datagridLoadSuccess">
				<thead>
					<tr>
						<th data-options="field:'id',hidden:true"></th>
						<th data-options="field:'built_in',hidden:true"></th>
						<th data-options="field:'group_name'" width="100"><%=rb.getString("SheBeiZuMingCheng")%></th>
					</tr>
				</thead>
			</table>
		</div>
		<div region="south" data-options="border:false,height:65">
			<div class="windowButtonGroup" style="margin-right:10px;margin-top:10px;">
			    <a href="#" class="linkbutton linkbutton_trend"  onclick="moveCpeToGroup()"><span><%=rb.getString("QueDing")%></span></a>
				<a href="#" class="linkbutton linkbutton_nowanna" onclick="closeDefaultWindow();"><span><%=rb.getString("QuXiao")%></span></a>
			</div>
			
		</div>
	</div>
</div>