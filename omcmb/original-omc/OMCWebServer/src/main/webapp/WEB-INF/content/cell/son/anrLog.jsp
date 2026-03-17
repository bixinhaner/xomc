<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<div class="easyui-layout" data-options="fit:true,border:false">
		<div data-options="region:'north',border:false,collapsible:false" style="margin-botom:20px;">
		        <div class="omcPageTitleDiv" style="padding-bottom:20px;">
					<ul class="omcPageTitleContainer">
						<li class="default"><%=rb.getString("ANR")%></li>
					</ul>
				</div>
		</div>
	<div region="center"  data-options="border:false" style="padding: 0 15px;">
		<table id="tableAnrLog" class="easyui-datagrid" fit="true" data-options="border:false,rownumbers:true,pagePosition:'bottom',striped:true,
			url:'${ctx}/SON/getAnrLoglist.action',pagination:true,idField:'ID',pageSize:'50',fitColumns:true,singleSelect:true,onLoadSuccess:datagridLoadSuccess,
		 	onLoadError: function(){
                $.messager.show({
                    title: '<%=rb.getString("TiShi")%>',
                    msg: '<%=rb.getString("JiaZaiShiBai")%>',
                    timeout: 3000
                });
            }">
		    <thead>
			<tr>
				<th data-options="field:'ID',hidden:true">ID</th>
				<th data-options="field:'MSG_TYPE'" width="90">MSG_TYPE</th>
				<th data-options="field:'PLMNID_A'" width="90">PLMNID_A</th>
				<th data-options="field:'CELL_IDENTIFIER_A'" width="100">CID_A</th>
				<th data-options="field:'PCI_A'" width="90">PCI_A</th>
				<th data-options="field:'HNBNAME_A'" width="120">HNBNAME_A</th>
				<th data-options="field:'PLMNID_B'" width="100">PLMNID_B</th>
				<th data-options="field:'CID_B'" width="90">CID_B</th>
				<th data-options="field:'PHY_CELLID_B'" width="100">PHY_CELLID_B</th>
				<th data-options="field:'TAC_B'" width="100">TAC_B</th>
				<th data-options="field:'EUTRA_CARRIER_DLARFCN_B'" width="250">EUTRA_CARRIER_DLARFCN_B</th>
<!-- 				<th data-options="field:'EUTRA_CARRIER_ULARFCN_B'" width="200">EUTRA_CARRIER_ULARFCN_B</th>
				<th data-options="field:'Dl_BANDWIDTH_B'" width="150">Dl_BANDWIDTH_B</th>
				<th data-options="field:'Ul_BANDWIDTH_B'" width="150">Ul_BANDWIDTH_B</th> -->
				<th data-options="field:'TIME'" width="150">TIME</th>
				<th data-options="field:'DEAL_RESULT'" width="500">DEAL_RESULT</th>
			</tr>
		    </thead>
		</table>
	</div>
</div>

<div id="toolbar_tableAnrLog" class="admin_query_head">
</div>