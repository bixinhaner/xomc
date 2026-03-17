<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<div id="winModifyPCIValue" style="width: 100%;height: 100%;">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false,fit:true" style="padding: 20px 10px;">
			<form id="modifyPCIForm">
			    <span><%=rb.getString("ShouDongXiuGaiPCI")%></span>
			    <div style="border-style: none;height: 40px;margin-top:10px">
				    <lable for="txt_N_NewPCI">Neighbor New PCI:</lable>
				    <input id="txt_N_NewPCI" type="text" name="suggested_newPCI" class="border border-box"/>
			    </div>
			</form>
		</div>
		<div region="south" data-options="border:true,height: 37" style="padding:5px 10px;border-width: 1px 0 0 0">
			<a class="easyui-linkbutton" onclick="closeWinPCIModification()" style="float:right;"><%=rb.getString("QuXiao")%></a>
			<a class="easyui-linkbutton" onclick="modifyNewPCIValue()" style="margin-right: 10px;float:right;"><%=rb.getString("QueDing")%></a>
		</div>
	</div>
</div>