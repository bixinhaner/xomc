<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<!-- 窗口-删除运营商 -->
<div id="winDelOperator" style="width: 100%;height: 100%;">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="padding:20px;">
			<div style="min-height:50px;">
				<%=rb.getString("QueRenShanChuYunYingShang")%>
			</div>
			<div style="text-align: center;">
				<input id="delOperatorCode" type="hidden" value=""/>
				<input id="adminPassword" type="password" style="width:100%;" placeholder="<%=rb.getString("QingShuRuMiMa")%>" class="easyui-validatebox border border-box item"/>
			</div>
		</div>
		<div region="south" data-options="border:false,height:71" style="padding:10px 20px 20px;">
			<div class="right">
				<span class="el-button el-button-primary" onclick="delOperator()"><%=rb.getString("QueDing")%></span>
				<span class="el-button" onclick="closeDefaultWindow()"><%=rb.getString("QuXiao")%></span>
			</div>
		</div>
	</div>
</div>