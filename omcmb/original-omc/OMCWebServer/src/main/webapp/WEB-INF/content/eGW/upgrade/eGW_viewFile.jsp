<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<style>
	.detailMesDiv div{
		display:inline-block;
	}
	#egwViewUpgradeFileInfo label{
		margin-bottom:5px;
		display:block;
	}
</style>
<div id="egwViewUpgradeFileInfo" style="display:flex;flex-direction:column;height:100%;">
		<div class='el-card__header'>
			<span><%=rb.getString("WenJianXinXi")%></span>
			<span class='el-icon el-icon-close' onclick='closeViewegwFileWindow()'></span>
		</div>
		<div class='el-card__body' style='display:flex;flex-direction:column;flex:1 1 auto;height:100%;overflow:auto;'>
			<div class='detailMesDiv'  style="background:#fff;border:1px solid #EEE">
			    <div style='margin:20px;'>
				    <div>
						<label style="width: 120px;"><%=rb.getString("WenJianMing")%></label>
						<input value='${fileName }' type="text" class="border border-box file_info" readonly style="width: 350px;"/>
					</div>
				    <div style='margin-left:40px;'>
				    	<label style="width: 120px;"><%=rb.getString("ChanPinLeiXingBiaoZhi")%></label>
						<input value='${productType }' readonly  type="text" class="border border-box file_info required" maxlength=100 style="width: 350px;height:26px;" />
					</div>
					<div style='margin-top:40px;'>
						<label style="width: 120px;"><%=rb.getString("BanBen")%></label>
						<input readonly  value='${version }' type="text" class="border border-box file_info required" maxlength=45 style="width: 350px;height:26px;"/>
					</div>
					<div style="margin-right:35px;margin-top:40px;" id="descViewBox">
						<label style="width: 120px; vertical-align: top;"><%=rb.getString("MiaoShu")%></label>
						<textarea value='' readonly cols="20" style="outline:none;padding-top:5px;font-size: 12px;width: 766px; height: 377px; resize: none;" rows="5"
							maxlength=500 class="border-box border file_info">${description }</textarea>
					</div>
				</div>
			</div>
		</div>
		
</div>
