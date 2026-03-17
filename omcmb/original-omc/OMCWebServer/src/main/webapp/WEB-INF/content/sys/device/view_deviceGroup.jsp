<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<%-- 查看设备组信息框 --%>
<div id="winViewDeviceGroup" style="width: 100%;height: 100%;">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="padding: 20px;">
			<div class="itemDiv">
				<span style="width: 150px"><%=rb.getString("SheBeiZuMingCheng")%></span>
				<input type="text" name="deviceGroup_name"  class="border border-box item" disabled="disabled" style="width:280px">
			</div>
			<div class="itemDiv">
				<span style="width: 150px"><%=rb.getString("ChuangJianZhe")%></span>
				<input type="text" name="deviceGroup_creator"  class="border border-box item" disabled="disabled" style="width: 280px;">
			</div>
			<div class="itemDiv">
				<span style="width: 150px"><%=rb.getString("ChuangJianShiJian")%></span>
				<input type="text" name="deviceGroup_createTime"  class="border border-box item" disabled="disabled" style="width: 280px;">
			</div>
			<div class="itemDiv">
				<span style="vertical-align: top; width: 150px;"><%=rb.getString("MiaoShu")%></span>
				<textarea rows="4" cols="20" name="deviceGroup_desc" class="border border-box item" style="height: 100px; resize: none; width: 280px;" disabled="disabled"></textarea>
			</div>
		</div>
	</div>
</div>
<script>

</script>