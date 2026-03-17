<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp" %>
<style>

</style>

<%-- 重启任务 --%>
<div class="panelDefault" style="display:flex;background:#E6E9F0;">
	<iframe id="dhcpRestart" style="width:100%;height:100%;border:none;" name="dhcpServerConfig" src="">
	
	</iframe>
</div>

<script type="text/javascript">
	var dhcpurlRestart = "${dhcpUrl}";
	$(function(){
		closeLoading();
		$("#dhcpRestart").attr("src",dhcpurlRestart+"/restart.do?isOMC=true");
	})
</script>