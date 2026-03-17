<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<%-- 窗口-修改基站信息（经度，维度）--%>
<div id="winModDeviceInfosForCpe" style="width: 100%;height: 100%;">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="center" data-options="border:false" style="padding: 20px;">
			<div class="itemDiv">
				<span><%=rb.getString("LONGITUDE")%></span>
				<input id="longitude_win_update" type="text" name="longitude_win" class="border border-box item" onblur="validateMaxAndMinVal_double(event)" min_value=-180 max_value=180
				 title="<%=rb.getString("ZuiXiaoZhi")%>: -180<%=rb.getString("DouHao")%><%=rb.getString("ZuiDaZhi")%>: 180">
			</div>
			<br/>
			<div class="itemDiv">
				<span><%=rb.getString("LATITUDE")%></span>
				<input id="latitude_win_update" type="text" name="latitude_win" class="border border-box item" onblur="validateMaxAndMinVal_double(event)" min_value=-90 max_value=90
				title="<%=rb.getString("ZuiXiaoZhi")%>: -90<%=rb.getString("DouHao")%><%=rb.getString("ZuiDaZhi")%>: 90">
			</div>
		</div>
		<div region="south" data-options="border:true,height:37" style="border-width: 1px 0 0 0; padding: 5px 10px;">
			<a href="#" class="easyui-linkbutton" style="float: right;" onclick="closeDefaultWindow()"><%=rb.getString("QuXiao")%></a>
			<a href="#" class="easyui-linkbutton" style="margin-right: 10px; float: right;" onclick="saveDeviceInfoForCpe()"><%=rb.getString("QueDing")%></a>
		</div>
	</div>
</div>