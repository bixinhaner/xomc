<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<div class="easyui-layout" data-options="border:false,fit:true">
	<div region="center" data-options="border:false">
		<table id="tableItfnAlarm" class="easyui-datagrid" data-options="
				url: '${ctx}/cell/fault/queryItfnAlarmList.action?timeZone='+timeZone,
			    striped: true,
			    pagination: true,
			    rownumbers: true,
			    border: false,
			    fit: true,
			    fitColumns: true,onLoadSuccess:datagridLoadSuccess">
        <thead>
        <tr>
            <th data-options="field:'OCCUR_TIME'" width="100"><%=rb.getString("GaoJingFaShengShiJian")%></th>
            <th data-options="field:'CLEAR_TIME'" width="100"><%=rb.getString("GaoJingQingChuShiJian")%></th>
            <th data-options="field:'STATUS'" width="100"><%=rb.getString("ZhuangTai")%></th>
        </tr>
        </thead>
    	</table>
	</div>
</div>