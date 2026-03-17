<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<%-- 窗口-选择移动到的设备组 --%>
<div id="winSelectGroupToMoveUps" style="width: 100%;height: 100%;">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="padding: 20px;">
			<table class="easyui-datagrid" id="upsSelectGroupToMove"
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
		<div region="south" data-options="border:false,height:57">
			<div class="windowButtonGroup" style="margin-right:20px">
				<a href="#" class="linkbutton linkbutton_trend"  onclick="moveToUpsGroup()"><span><%=rb.getString("QueDing")%></span></a>
				<a href="#" class="linkbutton linkbutton_nowanna"  onclick="closeDefaultWindow();"><span><%=rb.getString("QuXiao")%></span></a>   
			</div>
		</div>
	</div>
</div>