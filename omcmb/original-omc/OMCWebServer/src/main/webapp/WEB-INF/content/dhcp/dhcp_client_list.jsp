
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp" %>
<style>
</style>

<div class="panelDefault" style="display:flex;background:#E6E9F0;">
	<iframe id="dhcpClient" style="width:100%;height:100%;border:none;" name="dhcpServerConfig" src="">
	
	</iframe>
</div>

<script type="text/javascript">
var dhcpurlClient = "${dhcpUrl}";
	$(function(){
		closeLoading();
		$("#dhcpClient").attr("src",dhcpurlClient+"/queryIpDistributionList.do?isOMC=true&timeZone="+timeZone);
	})
</script>