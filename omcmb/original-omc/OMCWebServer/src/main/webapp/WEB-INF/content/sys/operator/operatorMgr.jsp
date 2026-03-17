<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%-- 运营商管理页面-分tab页显示“运营商信息”和“运营商小站信息” --%>

<div id="operatorMgr" class="easyui-tabs" data-options="fit:true,border:false">
	<div title="<%=rb.getString("YunYingShang")%>" data-options="href:'${ctx}/system/operator/goOperatorInfo.action'"></div>
	<div title="<%=rb.getString("YunYingShangXiaoZhanWeiHu")%>" data-options="href:'${ctx}/system/operator/goOperatorCell.action'"></div>
</div>