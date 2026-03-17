<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<style>
	#compactParamConfig_elfcell{
		position:relative;
		height:80vh;
	}
	.cellSetting_basic_header{
	    width: 100%;
	    line-height: 50px;
	    height: 50px;
	    color: #7993B6;
	    font-size: 16px;
	}
	.clearBoth{
		float:none !important;
		clear:both;
		margin-right:400px !important;
	}
	.itemDiv{
		width:375px;
		height:92px;
		float:left;
		margin-right:45px;
	} 
	.promptTitle{
		height:26px;
		line-height:26px;
		min-width:50px;
		font-size:12px;
		color:#9FB318;
		display : none;
	}
	.errorTitle{
		height:26px;
		line-height:26px;
		min-width:50px;
		font-size:12px;
		color:red;
		display : none;
	}
	#compactParamConfig_elfcell .itemDiv input,#compactParamConfig_elfcell .itemDiv select{
		width:350px;
	}
	#compactParamConfig_elfcell .itemDiv img{
		position:relative;
		top:4px;
		left:5px;
	}

</style>
<!-- header -->
<div class="cellSetting_basic_header">
	<div class="cellSettingTitle_basic" style="display:inline-block;margin-left:30px;"><%=rb.getString("SheZhi")%></div>
	<a class="titleIcon_close iconSize" style="float:right;margin-top:15px;margin-right:10px;" onclick="cellSettingCancel()"></a>
</div>
<!-- tab选项卡 -->
<div id="compactParamConfig_elfcell" class="inputInfos">
		<!-- 选项 -->
		<div class="omcPageTitleDiv omcLogLists">
			<ul class="omcPageTitleContainer titleTabsList">
				<li tabtit="wirelessSetting" onclick="turnTabs(this)" class="active"><%=rb.getString("WuXianSheZhi")%></li>
				<%-- <li tabtit="intentSetting" onclick="turnTabs(this)"><%=rb.getString("WangLuoSheZhi")%></li>
				<li tabtit="ICICSetting" onclick="turnTabs(this)"><%=rb.getString("ICICSheZhi")%></li>
				<li tabtit="NTPSetting" onclick="turnTabs(this)"><%=rb.getString("NTPSheZhi")%></li>
				<li tabtit="IPSECSetting" onclick="turnTabs(this)"><%=rb.getString("IpsecSheZhi")%></li>
				<c:if test="${lgw == 1}">	
					<li tabtit="LGWSetting" onclick="turnTabs(this)"><%=rb.getString("LGWSheZhi")%></li>
				</c:if> --%>
				
			</ul>
		</div>

		<!-- 选项对应内容-->
	
		<div class="omcTabsPage logsTabsMainPage" style="left:20px;top:45px;overflow-x:hidden;overflow-y:auto;height:60vh;">
			<%-- 无线设置面板 --%>
			<div  class="wirelessSetting" style="padding: 20px 0 0 20px;display:block">
				<div class="itemDiv">
					<span title="<%=rb.getString("RSECI")%>"><%=rb.getString("JiZhanID")%></span>
					<input id="ELFCELL_LTE_CELL_ECI" type="text" name="LTE_CELL_ECI" class="border border-box item" oldValue="${CELL_IDENTITY}" value="${CELL_IDENTITY}"
							onblur="validateMaxAndMinVal1_elfcell(event)" min_value="0" max_value="268435455" 
							title="int, min value: 0, max value: 268435455"/>
					<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
					<div class="errorTitle" id="ELFCELL_LTE_CELL_ECI_err">int, min value: 0, max value: 268435455</div>
				</div>
				<div class="itemDiv">
					<span title="<%=rb.getString("RSCellName")%>"><%=rb.getString("HostName")%></span>
					<input type="text" name="LTE_HOME_NODEB_NAME" id="ELFCELL_LTE_HOME_NODEB_NAME" class="border border-box item" oldValue="${HOST_NAME}" value="${HOST_NAME}"
							onblur="validateHostName_elfcell(event)" max_length="48"
							title="<%=rb.getString("SheBeiMingChengGuiZe")%>"/>
					<div class="errorTitle" id="ELFCELL_LTE_HOME_NODEB_NAME_err"><%=rb.getString("SheBeiMingChengGuiZe")%></div>
				</div>
				<div class="itemDiv" style="float:none;clear:both">
					<span title="<%=rb.getString("RSMME")%>"><%=rb.getString("HeXinWang")%></span>
					<input type="text" name="LTE_SIGLINK_SERVER_LIST" id="ELFCELL_LTE_SIGLINK_SERVER_LIST" class="border border-box item" 
						oldValue="" value="" onblur="validateIPAddress11_elfcell(event)" 
						title="<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>"/>
					<a onclick="addMMEIpInputText_elfcell(this)">
						<img src="${ctx}/skin/${manufacturer}/images/bi/setting_add.png"/>  
					</a>
					<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
					<div class="errorTitle" id="ELFCELL_LTE_SIGLINK_SERVER_LIST_err"><%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%></div>
				</div>
				<div class="itemDiv">
					<span title="<%=rb.getString("RSPLMN")%>"><%=rb.getString("PLMN")%></span>
					<input type="text" name="LTE_OAM_PLMNID" id="ELFCELL_LTE_OAM_PLMNID" class="border border-box item" oldValue="${PLMNID}" value="${PLMNID}"
							onblur="validateMaxAndMinVal1_elfcell(event)" min_value="10000" max_value="999999" 
							title="<%=rb.getString("PLMNTitle")%>"/>
					<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>	
					<div class="errorTitle" id="ELFCELL_LTE_OAM_PLMNID_err"><%=rb.getString("PLMNTitle")%></div>
				</div>
				<div class="itemDiv pciisLock">
					<span title="<%=rb.getString("RSPCI")%>"><%=rb.getString("PCI2")%></span>
					<input type="text" name="LTE_PHY_CELLID_LIST" id="ELFCELL_LTE_PHY_CELLID_LIST" class="border border-box item" oldValue="${PHYCELLID}" value="${PHYCELLID}"
							onblur="commonChangeTip_elfcell(event)" 
							vali-regex="/^(?:(?:(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3]),)*(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3]))$|^(?:(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3])\.\.(?:[0-9]|[1-9][0-9]|[1-4][0-9][0-9]|50[0-3]))$/"
							title="For example: '23' or '1,2,3' or '300..500' , and every number is between 0 and 503"/>
					<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>	
					<div class="errorTitle" id="ELFCELL_LTE_PHY_CELLID_LIST_err">int, min value: 0, max value: 503</div>
				</div>
				<div class="itemDiv">
					<span title="<%=rb.getString("RSTAC")%>"><%=rb.getString("TAC")%></span>
					<input type="text" name="LTE_TAC"  id="ELFCELL_LTE_TAC" class="border border-box item" oldValue="${TAC}" value="${TAC}"
							onblur="validateMaxAndMinVal1_elfcell(event)" min_value="0" max_value="65536"
							title="int, min value: 0, max value: 65536"/>
					<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>	
					<div class="errorTitle" id="ELFCELL_LTE_TAC_err">int, min value: 0, max value: 65536</div>	
				</div>
				<div class="itemDiv" style="display: none">
					<span title="<%=rb.getString("RSFrequencyBand")%>"><%=rb.getString("ZhiChiPinDuan")%></span>
					<input type="text" name="LTE_BANDS_SUPPORTED" id="ELFCELL_LTE_BANDS_SUPPORTED" class="border border-box item" 
					 		oldValue="${BANDS_SUPPORTED}" value="${BANDS_SUPPORTED}" 
					 		onblur="commonChangeTip_elfcell(event)" title="stringList-[1:62]"/>
					<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>	
					<div class="errorTitle" id="ELFCELL_LTE_BANDS_SUPPORTED_err">stringList-[1:62]</div>
				</div>
				<div class="itemDiv pciisLock">
					<span title="<%=rb.getString("RSEarfcn")%>"><%=rb.getString("PinDian")%></span>
					<input type="text" name="LTE_UL_DL_EARFCN" id="ELFCELL_LTE_UL_DL_EARFCN" class="border border-box item" oldValue="${EARFCNDLINUSE}" value="${EARFCNDLINUSE}"
							onblur="commonChangeTip_elfcell(event);translateToFre(this)"  onfocus="translateToEarfen(this)"
							vali-regex="/^(?:(?:(?:[0-9]|[1-9][0-9]|[1-9][0-9][0-9]|[1-9][0-9][0-9][0-9]|[1-5][0-9][0-9][0-9][0-9]|6[1-4][0-9][0-9][0-9]|65[0-4][0-9][0-9]|655[0-2][0-9]|6553[0-5]),)*(?:[0-9]|[1-9][0-9]|[1-9][0-9][0-9]|[1-9][0-9][0-9][0-9]|[1-5][0-9][0-9][0-9][0-9]|6[1-4][0-9][0-9][0-9]|65[0-4][0-9][0-9]|655[0-2][0-9]|6553[0-5]))$|^(?:(?:[0-9]|[1-9][0-9]|[1-9][0-9][0-9]|[1-9][0-9][0-9][0-9]|[1-5][0-9][0-9][0-9][0-9]|6[1-4][0-9][0-9][0-9]|65[0-4][0-9][0-9]|655[0-2][0-9]|6553[0-5])\.\.(?:[0-9]|[1-9][0-9]|[1-9][0-9][0-9]|[1-9][0-9][0-9][0-9]|[1-5][0-9][0-9][0-9][0-9]|6[1-4][0-9][0-9][0-9]|65[0-4][0-9][0-9]|655[0-2][0-9]|6553[0-5]))$/"
							title="For example: '23' or '1,2,3' or '300..65535' , and every number is between 0 and 65535"/>
							<!-- onblur="validateMaxAndMinVal1_elfcell(event)" title="intList-[0:65535]" min_value="0" max_value="65535"/> -->
					<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>	
					<div class="errorTitle" id="ELFCELL_LTE_UL_DL_EARFCN_err">int, min value: 0, max value: 65535</div>
				</div>
				<div class="itemDiv">
					<span title="<%=rb.getString("RSReferenceSignalPower")%>"><%=rb.getString("CanKaoXinHaoGongLv")%></span>
						<input type="text" name="LTE_REFERENCE_SIG_POWER" id="ELFCELL_LTE_REFERENCE_SIG_POWER" class="border border-box item" 
							oldValue="${REFERENCE_SIGNAL_POWER}" value="${REFERENCE_SIGNAL_POWER}"
							onblur="commonChangeTip_elfcell(event)" 
							vali-regex="/^(?:(?:(?:-*[0-9]|-*[1-4][0-9]|-5[0-9]|-60|50),)*(?:-*[0-9]|-*[1-4][0-9]|-5[0-9]|-60|50))$|^(?:(?:-*[0-9]|-*[1-4][0-9]|-5[0-9]|-60|50)\.\.(?:-*[0-9]|-*[1-4][0-9]|-5[0-9]|-60|50))$/"
							title="For example: '23' or '1,2,3' or '-30..50',int, min value: -60, max value: 50" />
						<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>	
					<div class="errorTitle" id="ELFCELL_LTE_REFERENCE_SIG_POWER_err">int, min value: -60, max value: 50</div>
				</div>
				<div class="itemDiv" style="display:none">
					<span title="<%=rb.getString("RSDLBandWidth")%>"><%=rb.getString("XiaXingDaiKuan")%></span>
					<select name="LTE_DL_BANDWIDTH"  id="ELFCELL_LTE_DL_BANDWIDTH" class="border border-box item">
						<option value=""> </option>
						<option value="n6">CELL_BW_N6(1.4M)</option>
						<option value="n15">CELL_BW_N15(3M)</option>
						<option value="n25">CELL_BW_N25(5M)</option> 
						<option value="n50">CELL_BW_N50(10M)</option>
						<option value="n75">CELL_BW_N75(15M)</option>
						<option value="n100">CELL_BW_N100(20M)</option>
					</select>
					<input type="hidden" value="" id="ELFCELL_OLD_LTE_DL_BANDWIDTH">
					
				</div>
				
				<div class="itemDiv" style="display:none">
					<span title="<%=rb.getString("RSBandwidth")%>"><%=rb.getString("DaiKuan")%></span>
					<select name="LTE_UL_BANDWIDTH" id="ELFCELL_LTE_UL_BANDWIDTH" class="border border-box item">
						<option value=""> </option>
						<option value="n6">CELL_BW_N6(1.4M)</option>
						<option value="n15">CELL_BW_N15(3M)</option>
						<option value="n25">CELL_BW_N25(5M)</option>
						<option value="n50">CELL_BW_N50(10M)</option>
						<option value="n75">CELL_BW_N75(15M)</option>
						<option value="n100">CELL_BW_N100(20M)</option>
					</select>
					<input type="hidden" value="" id="ELFCELL_OLD_LTE_UL_BANDWIDTH">
				</div>
				<div class="itemDiv">
					<span title="<%=rb.getString("RSSfAssignment")%>"><%=rb.getString("ZiZhenPeiBi")%></span>
					<select name="LTE_TDD_SUBFRAME_ASSIGNMENT" id="ELFCELL_LTE_TDD_SUBFRAME_ASSIGNMENT" class="border border-box item">
						<option value=""> </option>
						<option value="0">SA0</option>
						<option value="1">SA1</option>
						<option value="2">SA2</option>
						<option value="3">SA3</option>
						<option value="4">SA4</option>
						<option value="5">SA5</option>
						<option value="6">SA6</option>
					</select>
					<input type="hidden" value="" id="ELFCELL_OLD_LTE_TDD_SUBFRAME_ASSIGNMENT">
					<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
				</div>
				<div class="itemDiv">
					<span title="<%=rb.getString("RSSpecialSfPatterns")%>"><%=rb.getString("TeShuZiZhenPeiBi")%></span>
					<select name="LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS" id="ELFCELL_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS" class="border border-box item">
						<option value=""> </option>
						<option value="0">SSP0</option>
						<option value="1">SSP1</option>
						<option value="2">SSP2</option>
						<option value="3">SSP3</option>
						<option value="4">SSP4</option>
						<option value="5">SSP5</option>
						<option value="6">SSP6</option>
						<option value="7">SSP7</option>
						<option value="8">SSP8</option>
					</select>
					<input type="hidden" value="" id="ELFCELL_OLD_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS">
					<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>
				</div>
				<div class="itemDiv">
					<span title="<%=rb.getString("RSRootSequenceIndex")%>"><%=rb.getString("GenXuLieSuoYin")%></span> <input type="text" name="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST"
						id="ELFCELL_LTE_ROOT_SEQ_INDEX" class="border border-box item"
						oldValue="${ROOT_SEQUENCE_INDEX}" value="${ROOT_SEQUENCE_INDEX}"
						onblur="commonChangeTip_elfcell(event)"
						vali-regex="/^(?:(?:(?:[0-9]|[1-9][0-9]|[1-7][0-9][0-9]|8[0-3][0-7]),)*(?:[0-9]|[1-9][0-9]|[1-7][0-9][0-9]|8[0-3][0-7]))$|^(?:(?:[0-9]|[1-9][0-9]|[1-7][0-9][0-9]|8[0-3][0-7])\.\.(?:[0-9]|[1-9][0-9]|[1-7][0-9][0-9]|8[0-3][0-7]))$/"
						title="For example: '23' or '1,2,3' or '30..50',int, min value: 0, max value: 837" />
						<div class="promptTitle"><span><%=rb.getString("SheZhiCellId")%></span><span class="unmodiValue"></span></div>	
						<div class="errorTitle" id="ELFCELL_LTE_ROOT_SEQ_INDEX_err">int, min value: 0, max value: 837</div>
				</div>
			</div>
			<%-- 网络设置面板 --%>
			<%-- NTP设置面板 --%>
			<%-- IPSEC设置面板 --%>
		</div>
	
		<div class="windowButtonGroup" style="position:absolute;bottom:37px;left:47px;">
			<a class="easyui-linkbutton linkbutton linkbutton_trend" onclick="cellSettingCommit_elfcell()"><span><%=rb.getString("QueDing")%></span></a>
			<a class="easyui-linkbutton linkbutton linkbutton_nowanna" onclick="cellSettingCancel()"><span><%=rb.getString("QuXiao")%></span></a>
		</div>
	
</div>

<%-- 窗口-右键设置进度条 --%>
<div id="winSettingPro_elfcell" title="<%=rb.getString("CanShuPeiZhiJinDu")%>" class="easyui-window"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:400,height:70,resizable:false,closable:false">
    <img src="${ctx}/js/jquery-easyui/themes/bootstrap/images/loading.gif"/>
    <span style="margin-left:75px;"><%=rb.getString("eNodeBZhengZaiSheZhi")%></span>
   
</div>

<script type="text/javascript">
$("select[name='LTE_UL_BANDWIDTH']").bind("change",function(){
	//移除之前被选中的元素
	$("select[name='LTE_DL_BANDWIDTH'] option:selected").removeAttr("selected");
	//选中新选择的元素
	var opts = $("select[name='LTE_DL_BANDWIDTH']").find("option");
	var oldValue=$("#ELFCELL_OLD_LTE_UL_BANDWIDTH").val();
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == this.value){
			$(opts[i]).attr("selected",true);
			if(oldValue!=this.value){
				$("#ELFCELL_LTE_UL_BANDWIDTH").css({"color":"blue"});
				$("#ELFCELL_LTE_UL_BANDWIDTH option").css({"color":"black"});
				/* $.map($("#ELFCELL_LTE_UL_BANDWIDTH option:not(:selected)"),function(ele){
					$(this).css({"color":"black"});
				}).join(","); */
			}else{
				$("#ELFCELL_LTE_UL_BANDWIDTH").css({"color":"black"});	
			}
			break;
		}
	}
});


$("select[name='LTE_TDD_SUBFRAME_ASSIGNMENT']").bind("change",function(){
	//移除之前被选中的元素
	/* $("select[name='LTE_TDD_SUBFRAME_ASSIGNMENT'] option:selected").removeAttr("selected");  */
	//选中新选择的元素
	var opts = $("select[name='LTE_TDD_SUBFRAME_ASSIGNMENT']").find("option");
	var oldValue=$("#ELFCELL_OLD_LTE_TDD_SUBFRAME_ASSIGNMENT").val();
	var ele = $(this);
	var oldValueText = "";
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == oldValue){
			oldValueText = $(opts[i]).text();
		}
	}
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == this.value){
			$(opts[i]).attr("selected",true);
			if(oldValue!=this.value){
				$("#ELFCELL_LTE_TDD_SUBFRAME_ASSIGNMENT").css({"color":"blue"});
				$("#ELFCELL_LTE_TDD_SUBFRAME_ASSIGNMENT option").css({"color":"black"});
				ele.siblings(".promptTitle").show();
		    	ele.siblings(".promptTitle").find(".unmodiValue").text(oldValueText);
			}else{
				$("#ELFCELL_LTE_TDD_SUBFRAME_ASSIGNMENT").css({"color":"black"});	
				ele.siblings(".promptTitle").hide();
			}
			break;
		}
	}
	
});


$("select[name='LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS']").bind("change",function(){
	//移除之前被选中的元素
	/* $("select[name='LTE_TDD_SUBFRAME_ASSIGNMENT'] option:selected").removeAttr("selected");  */
	//选中新选择的元素
	var opts = $("select[name='LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS']").find("option");
	var oldValue=$("#ELFCELL_OLD_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS").val();
	var ele = $(this);
	var oldValueText = "";
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == oldValue){
			oldValueText = $(opts[i]).text();
		}
	}
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == this.value){
			$(opts[i]).attr("selected",true);
			if(oldValue!=this.value){
				$("#ELFCELL_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS").css({"color":"blue"});
				$("#ELFCELL_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS option").css({"color":"black"});
				ele.siblings(".promptTitle").show();
		    	ele.siblings(".promptTitle").find(".unmodiValue").text(oldValueText);
			}else{
				$("#ELFCELL_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS").css({"color":"black"});
				ele.siblings(".promptTitle").hide();
			}
			break;
		}
	}
	
});

<%-- 加载完成事件 --%>
$(function() {
	
	closeLoading();
	//加载完成频点转频率格式化
	var elfearoldVal = "${EARFCNDLINUSE}";
	var elfearfcnVal = translateToFre(elfearoldVal);
	$("#ELFCELL_LTE_UL_DL_EARFCN").val(elfearfcnVal);
	
	/*  判断CPE下的PCI是否锁定 */
	//如果此基站有CPE进行锁频，那么对应的pci，bandclass,earfcn3个输入框将变为不可编辑
	var PCIIsLock ="${hasCpeFreqLock}";
	if(!PCIIsLock){
		$('.pciisLock img').attr("src","${ctx}/skin/${manufacturer}/images/bi/pci_unlock.png");
	}else{
		$('.pciisLock img').attr("src","${ctx}/skin/${manufacturer}/images/bi/pci_lock.png");
		$('.pciisLock img').attr('title','This device is locked')
		$('.pciisLock input').attr('disabled','disabled')
	}
	//如果SAS开关打开，则该参数不可配置
	if(SASEnble == "1"){
		$("#ELFCELL_LTE_UL_DL_EARFCN").attr("disabled",true);
		$("#ELFCELL_LTE_UL_BANDWIDTH").attr("disabled",true).css("background","#EAF1F4");
	}
	
	/* var isCpeFreqLock = "${hasCpeFreqLock}";
	//如果此基站有CPE进行锁频，那么对应的pci，bandclass,earfcn3个输入框将变为不可编辑
	if (isCpeFreqLock == 1) {
		$("input[name='LTE_PHY_CELLID_LIST']").attr("disabled", true);
		//$("input[name='LTE_FREQ_BAND_INDICATOR']").attr("disabled", true);
		$("input[name='LTE_UL_DL_EARFCN']").attr("disabled", true);
	} */
	
	<%-- 为下拉框赋值 --%>
	$("select[name='LTE_DL_BANDWIDTH']").val("${DL_BANDWIDTH}");
	$("select[name='LTE_UL_BANDWIDTH']").val("${UL_BANDWIDTH}");
	$("select[name='LTE_TDD_SUBFRAME_ASSIGNMENT']").val("${SUBFRAME_ASSIGNMENT}");
	$("select[name='LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS']").val("${SPECIAL_SUBFRAME_PATTERNS}");
	//值为空表示不支持IP_PROTO参数
	/* if ("${IP_PROTO}" == "") {
		$("select[name='UNIT_IP_PROTO']").val("dhcp");
		$("select[name='UNIT_IP_PROTO']").attr("disabled","disabled");
		$("#ELFCELL_OLD_UNIT_IP_PROTO").val("dhcp");
	} else {
		$("select[name='UNIT_IP_PROTO']").val("${IP_PROTO}");
		$("#ELFCELL_OLD_UNIT_IP_PROTO").val("${IP_PROTO}");
	} */
	
	$("#ELFCELL_OLD_LTE_DL_BANDWIDTH").val("${DL_BANDWIDTH}");
	$("#ELFCELL_OLD_LTE_UL_BANDWIDTH").val("${UL_BANDWIDTH}");
	$("#ELFCELL_OLD_LTE_TDD_SUBFRAME_ASSIGNMENT").val("${SUBFRAME_ASSIGNMENT}");
	$("#ELFCELL_OLD_LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS").val("${SPECIAL_SUBFRAME_PATTERNS}");
	
	//初始化核心网地址输入框
	var mmeAddress = "${S1SIGLINKSERVERLIST}";
	if (mmeAddress != "") {
		var mmeArr = mmeAddress.split(",");
		//当mme地址为n且n>1时，需要重新生成n-1个输入框
		if (mmeArr.length > 1) {
			//初始化第一个输入框的值和oldValue值
			var firstMMEDiv = $("input[name='LTE_SIGLINK_SERVER_LIST']:first");
			firstMMEDiv.val(mmeArr[0]);
			firstMMEDiv.attr("oldValue", mmeArr[0]);
			
			var newMMEInputCount = mmeArr.length;
			//生成n-1个输入框
			for(var count = 1; count < newMMEInputCount; count++) {
				//添加新的输入框
				var $mmeSpan = $("<span style='display:inline-block;visibility:hidden'><%=rb.getString("HeXinWang")%></span>");
				var $mmeDiv = $("<div class='itemDiv'></div>");
				var $mmeInput = $("<input type='text' name='LTE_SIGLINK_SERVER_LIST' class='border border-box item' "
						+ "id='LTE_SIGLINK_SERVER_LIST_org_"+count+"' onblur='validateIPAddress11_elfcell(event)' oldValue='" + mmeArr[count] + "' value='" + mmeArr[count] + "' style='margin-left:4px'></input>");
				var $mmeButtn = $("<a onclick='removeMMEIpInputText_elfcell(this)' style='margin-left:3px'>" 
						+ "<img src='${ctx}/skin/${manufacturer}/images/bi/setting_sub.png'/> </a>");
				var $tip = $("<div class='promptTitle'><span><%=rb.getString("SheZhiCellId")%></span><span class='unmodiValue'></span></div>" + 
						"<div class='errorTitle' id='LTE_SIGLINK_SERVER_LIST_org_"+ count +"_err'><%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%></div>");
				
				$mmeDiv.append($mmeSpan);
				$mmeDiv.append($mmeInput);
				$mmeDiv.append($mmeButtn);
				$mmeDiv.append($tip);
				
				//将新增的输入框插入到plmn参数之前
				var plmnidDiv = $("input[name='LTE_OAM_PLMNID']").parent("div");
				$mmeDiv.insertBefore(plmnidDiv);
			}
		} else {
			var firstMMEDiv = $("input[name='LTE_SIGLINK_SERVER_LIST']:first");
			firstMMEDiv.val(mmeArr[0]);
			firstMMEDiv.attr("oldValue", mmeArr[0]);
		}
	}
});

<%-- 关闭窗口 --%>
function closeSelfWin_elfcell() {
	/* $("#winCellSettings").window("close"); */
	closeDefaultWindow();
}
<%-- 提交 --%>
function cellSettingCommit_elfcell() {
	var cellCode = "<%=request.getParameter("smallCellCode") %>";
	//判断基站是否连接
	$.post("${ctx}/cell/cpeinfos/getCellConnStatus.action", {"small_cell_code": cellCode}, function(data){
		// 基站未连接，不允许修改参数
		if(data["connStatus"] == "false") {
			$.messager.alert(TiShi, "<%=rb.getString("JiZhanWeiLianJie")%>");
			return;
		} else {
			if ($("#compactParamConfig_elfcell .item.err_border").length > 0) {
				return;
			}
			
			var paramsForeNodeBName = {};
			var params = {};
			//是否修改ip获取方式相关的参数
			var isChangeIpProto = false;
			
			$("#compactParamConfig_elfcell .item").each(function() {
				var paramName = $(this).attr("name");
				var newValue = $(this).val();
				var newValueFlag = newValue.indexOf("(");
				if(newValueFlag > -1){
					var newValueArr = newValue.split("(");
					newValue = newValueArr[0];
				}
				var oldValue = $(this).attr("oldValue");
				if(oldValue == undefined){
					//下拉框得参数为获取id为OLD_+name的value值
					var selectId = "ELFCELL_OLD_" + paramName;
					oldValue = $("#" + selectId).val();
				}
				//核心网地址参数特殊处理
				if (paramName == "LTE_SIGLINK_SERVER_LIST") {
					//如果是核心网地址且不为空，则将多个参数值拼接起来
					if (params[paramName] != null) {
						var mmeAdd = params[paramName];
						//确保不添加重复地址
						if (mmeAdd.indexOf(newValue) == -1) {
							params[paramName] =  params[paramName] + newValue + ","; 
						}
					} else {
						params[paramName] =  newValue + ","; 
					}
				/* } else if (paramName == "LTE_HOME_NODEB_NAME") {//配置基站名称需要特殊处理
					if (oldValue != newValue) {
						var paramsForCellName = {
								eNodeBName : newValue,
								small_cell_code : cellCode
						};
						$.post("${ctx}/cell/cpeinfos/updateCellName.action", paramsForCellName, function(data){
							$("#tableHomeCellList").datagrid("reload");
							if (!data["success"]) {
								$.messager.alert(TiShi, data["message"]);
							}
						}, "json");
						paramsForeNodeBName["changed"] = true;
					} */
				} else {
					//其他的参数值变化才加入到params中
					if (oldValue != newValue) {
						if (paramName == "LTE_UL_DL_EARFCN") {
							//上行频点和下行频点合并为频点，因此需要进行特殊处理
							params["LTE_DL_EARFCN"] = newValue;
							//params["LTE_UL_EARFCN"] = newValue;//高通平台基站暂不支持上行频点
						} else {
							params[paramName] =  newValue;
						}
					}
				}
			});
			//如果当前有设置的MME地址，去掉参数params["ELFCELL_LTE_SIGLINK_SERVER_LIST"]最后一个字符，
			if (params["LTE_SIGLINK_SERVER_LIST"] != null) {
				var tempMmeAddress = params["LTE_SIGLINK_SERVER_LIST"];
				var mmeAddress = tempMmeAddress.substring(0, tempMmeAddress.length - 1);
				if (mmeAddress == "${S1SIGLINKSERVERLIST}") {
					//删除params对象的LTE_SIGLINK_SERVER_LIST属性
					delete params.LTE_SIGLINK_SERVER_LIST;
				} else {
					params["LTE_SIGLINK_SERVER_LIST"] = mmeAddress;
				}
			}
			//判断当前修改的参数是否为空
			var isParamsEmpty = jQuery.isEmptyObject(params);
			var isparamsForCellName = jQuery.isEmptyObject(paramsForeNodeBName);
			//当前参数有修改才下发修改参数的命令
			if(!isParamsEmpty || !isparamsForCellName) {
				params["cell"] = cellCode;
				
				$("#winSettingPro_elfcell").window("open");
				//请求修改参数
				$.post("${ctx}/cell/param/compactParamConfig.action", params, function(data) {
					$("#winSettingPro_elfcell").window("close");
					if (data["success"]) {
						cellSettingCancel();
						$.messager.alert(TiShi, "<%=rb.getString("CaoZuoChengGong")%>");
						$("#tableHomeCellList").datagrid("reload");
					} else {
						$.messager.alert(TiShi, data["message"]);
					}
				}, "json");
				
			} else {
				$.messager.alert(TiShi, "<%=rb.getString("CanShuZhiMeiYouBianHua")%>");
				return;
			}
		}
	},"json");
}
var randomNum_elfcell = 0;
//添加一个MME地址输入框
function addMMEIpInputText_elfcell(e) {
	//添加新的输入框
	randomNum_elfcell += 1;
	var oldValue =$("input[id='ELFCELL_LTE_SIGLINK_SERVER_LIST']").first().attr("oldValue");
	var $mmeSpan = $("<span style='display:inline-block;visibility:hidden'><%=rb.getString("HeXinWang")%></span>");
	var $mmeDiv = $("<div class='itemDiv'></div>");
	var $mmeInput = $("<input type='text' name='LTE_SIGLINK_SERVER_LIST' class='border border-box item' "
		 + "id='LTE_SIGLINK_SERVER_LIST_"+randomNum_elfcell+"' onblur='validateIPAddress11_elfcell(event)'  oldValue='' style='margin-left:0px'></input>");
	var $mmeButtn = $("<a onclick='removeMMEIpInputText_elfcell(this)' style='margin-left:3px'>" 
			+ "<img src='${ctx}/skin/${manufacturer}/images/bi/setting_sub.png'/></a>");
	var $tip = $("<div class='promptTitle'><span><%=rb.getString("SheZhiCellId")%></span><span class='unmodiValue'></span></div>" + 
			"<div class='errorTitle' id='LTE_SIGLINK_SERVER_LIST_"+ randomNum_elfcell +"_err'><%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%></div>");
	
	$mmeDiv.append($mmeSpan);
	$mmeDiv.append($mmeInput);
	$mmeDiv.append($mmeButtn);
	$mmeDiv.append($tip);
	$mmeDiv.addClass("clearBoth");
	//将新增的输入框插入到plmn参数之前
	var plmnidDiv = $("input[name='LTE_OAM_PLMNID']").parent("div");
	$mmeDiv.insertBefore(plmnidDiv);
	
	//将当前输入框后面的图标改为删除图标，并重新绑定事件
	$(e).children("img").attr("src","${ctx}/skin/${manufacturer}/images/bi/setting_add.png");
	$(e).attr("onclick", "addMMEIpInputText_elfcell(this)");
}
//删除选中的MME地址输入框
function removeMMEIpInputText_elfcell(e) {
	$(e).parent("div").remove();
	//如果当前第一个MME输入框的span不显示，将其显示
	var mmeDiv = $("input[name='LTE_SIGLINK_SERVER_LIST']:first");
	var isVisible = mmeDiv.parent("div").children("span").css("visibility");
	if (isVisible == "hidden") {
		mmeDiv.parent("div").children("span").show();
	}
}

function validateMaxAndMinVal1_elfcell(e) {
    var ele = $(e["target"]);
    ele.siblings(".promptTitle").hide();
    validateMaxAndMinVal(e);
    if(!$("#" + ele.attr("id") + "_err").is(':visible')){
	    var currVal = ele.val();
	    var oldValue = ele.attr("oldValue");
	    if(currVal!=oldValue){
	    	$("#" +ele.attr("id")).css({"color":"blue"});
	    	ele.siblings(".promptTitle").show();
	    	ele.siblings(".promptTitle").find(".unmodiValue").text(oldValue);
	    }else{
	    	$("#" +ele.attr("id")).css({"color":"black"});
	    	ele.siblings(".promptTitle").hide();
	    }
	    var id = ele.attr("id");
	    if("1" != "${isElfCell}" && id == "ELFCELL_LTE_FREQ_BAND_INDICATOR"){
	    	$("#ELFCELL_LTE_BANDS_SUPPORTED").val(currVal);
	    }
    }
}

function validateHostName_elfcell(e){
	var ele = $(e["target"]);
    var currVal = ele.val();
    var reg = /^[A-Za-z0-9_\-]{0,48}$/;
    if(currVal!=""){
	    if(!reg.test(currVal)){
	    	 $("#" + ele.attr("id") + "_err").show();
	         ele.addClass("err_border");
	    }else{
	    	 $("#" + ele.attr("id") + "_err").hide();
	         ele.removeClass("err_border");
	    }
    }else{
         ele.removeClass("err_border");
    }
}
//mme输入框失去焦点事件
/* function validateIPAddress1_elfcell(e){
  	var ele = $(e["target"]);
    var currVal = ele.val();
    var oldValue = ele.attr("oldValue");
    if(currVal!=oldValue){
    	$("#" +ele.attr("id")).css({"color":"blue"});
    	ele.siblings(".promptTitle").show();
    	ele.siblings(".promptTitle").find(".unmodiValue").text(oldValue);
    }else{
    	$("#" +ele.attr("id")).css({"color":"black"});
    	ele.siblings(".promptTitle")..hide();
    }
    validateIPAddress(e);
} */

function validateIPAddress11_elfcell(e){
  	var ele = $(e["target"]);
    ele.siblings(".promptTitle").hide();
    validateIPAddress(e);
    if(!$("#" + ele.attr("id") + "_err").is(':visible')){
	    var currVal = ele.val();
	    var oldValue = ele.attr("oldValue");
	    if(currVal!=oldValue){
	    	$(ele).css({"color":"blue"});
	    	ele.siblings(".promptTitle").show();
	    	ele.siblings(".promptTitle").find(".unmodiValue").text(oldValue);
	    }else{
	    	$(ele).css({"color":"black"});
	    	ele.siblings(".promptTitle").hide();
	    }
    }
}

function commonChangeTip_elfcell(e){
    var ele = $(e["target"]);
    ele.siblings(".promptTitle").hide();
	validateByRegex(e)
	if(!$("#" + ele.attr("id") + "_err").is(':visible')){
	    var currVal = ele.val();
	    var oldValue = ele.attr("oldValue");
	    if(currVal!=oldValue){
	    	$("#" +ele.attr("id")).css({"color":"blue"});
	    	ele.siblings(".promptTitle").show();
	    	ele.siblings(".promptTitle").find(".unmodiValue").text(oldValue);
	    }else{
	    	$("#" +ele.attr("id")).css({"color":"black"});
	    	ele.siblings(".promptTitle").hide();
	    }
	}
}
//右键X号收起列表详情
function cellSettingCancel(){
	var settingInput = $("#cellsetting_basic input[type!=hidden]:visible");
	var settingselect = $("#cellsetting_basic select:visible");
	$.each(settingInput,function(index,ele){
		$(ele).val($(ele).attr('oldvalue'))  ;
	})
	$.each(settingselect,function(index,ele){
		$(ele).val($(ele).siblings('input:hidden').val());
	})
    $("#cellsetting_basic").animate({right:'-1200px'},500);
}
</script>