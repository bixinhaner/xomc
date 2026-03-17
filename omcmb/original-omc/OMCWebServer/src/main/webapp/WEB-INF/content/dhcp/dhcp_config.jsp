
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp" %>
<style>

</style>

<%-- 重启任务 --%>
<div class="panelDefault" style="display:flex;background:#E6E9F0;">
	<iframe id="dhcpServerConfig" style="width:100%;height:100%;border:none;" name="dhcpServerConfig" src="">
	
	</iframe>
</div>

<script type="text/javascript">
var dhcpurlsetting = "${dhcpUrl}"

	$(function(){
		closeLoading();
		$("#dhcpServerConfig").attr("src",dhcpurlsetting+"/setting.do?isOMC=true")
	})
</script>