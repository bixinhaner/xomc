<%@ page import="java.util.Locale" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>

<div style='display:flex;flex-direction:column;height:100%;'>
		<div class='el-card__header'>
			<span><%=rb.getString("XinXi")%></span>
			<span style="position: absolute;right: 20px;top: 0px;display: flex;align-items: center;" onclick='closeviewNewEPCTrace()'>
				<span class='el-button el-button--text' style="padding:10px 0px">
					<i class="titleIcon_close iconSize"></i>
				</span>
			</span>
		</div>
		
		<!-- <空白填充区域> -->
		<!-- <div style="width:924px;height:50px;"></div> -->
		<div class='el-card__body'>
			<!-- 信息填写部分 -->
			<div class="taskStatusContainer" style="position:relative;overflow:auto;height:670px;padding-bottom:15px;">
				<!-- 基本信息 -->
				<div class="group-title not-extend">
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
				</div>
				<div class="containerBox defaultInput">
					<p style="height:">
						<label><%=rb.getString("GenZongMingCheng")%></label>
						<input id="viewepcTaskTraceName" type="text" name="traceName" disabled="disabled"/>
						<span style="display:none;height:20px;line-height:20px;color:red;"><%=rb.getString("GenZongMingChengBuNengWeiKong")%></span>
					</p>
					<p style="margin-right:0px">
						<label><%=rb.getString("GenZongCanKaoHao")%></label>
						<input id="viewepcTaskTraceId" type="text" name="traceId" disabled="disabled"/>
					</p>
					<p style="margin-right:0px">
						<label><%=rb.getString("BeiZhuXinXi")%></label>
						<textarea id="viewepcTaskRemarks" style="width:350px;height:90px;border-color:#DEDFE6; name="remarkInfos" disabled="disabled"></textarea>
					</p>
				</div>
			
				<!-- 跟踪配置 -->
				<div class="group-title not-extend">
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("GenZongPeiZhi")%></span>
				</div>
				<div class="containerBox defaultInput">
					<div style="margin-bottom:25px;">
						<p>
							<label><%=rb.getString("GenZongSheBei")%></label>
							<input style="background:#F5F7FA" id="viewshowEPCTraceName" type="text" disabled="disabled" name="traceDevice" />
							<span class="showDevice"></span>
						</p>
					</div>
					
					<p style="margin-right:0px;margin-bottom:35px;height:82px">
						<label>IMSI</label>
						<input id="viewEPCTraceIMSI" type="text" name="traceIMSI" onblur="testIMSI(this)" disabled="disabled"/>
						<span style="display:none;height:20px;line-height:20px;color:red;">IMSI不能为空</span>
					</p>				
					<div class="containerBox defaultInput" style="border-bottom:none;margin-bottom:0px;margin-left:0px;padding-bottom:0px">
						<div style="display:block;margin-bottom:0px;">
							<label style="display:block;line-height:36px;font-size:12px;"><%=rb.getString("JieKouLeiXing")%></label>
							<div class="EPCInterface" style="width:600px;height:105px;padding-top:10px;border:1px solid #DEDFE6;background:#F5F7FA;">
								<div class="EPCChoseMME" style="width:350px;margin-bottom:30px;height:21px;padding-top:5px;">
									<label style="margin-right:20px;margin-left:10px" for="mmeNEType">MME:</label>
									<input class="alignCenter" id="viewEPCS1" type="checkbox" name="MMEType" value="S1" disabled="disabled">
									<label style="margin-right:55px;" for="viewEPCS1">S1</label>
									<input class="alignCenter" id="viewEPCS6a" type="checkbox" name="MMEType" value="HSS" disabled="disabled">
									<label for="viewEPCS6a">S6a</label>
								</div>
								<div class="EPCChoseHSS" style="width:350px;height:21px;padding-top:5px;">
									<label style="margin-right:26px;margin-left:10px" for="mmeNEType">HSS:</label>
									<input class="alignCenter" id="viewHssEPCS6a" type="checkbox" name="HSSType" value="S6a" disabled="disabled">
									<label style="margin-right:55px;" for="viewHssEPCS6a">S6a</label>
								</div>
							</div>
						</div>
					</div>
					
					
				</div>
			
			<!-- 执行方式 -->
				<div class="group-title not-extend">
					<span class="title-icon"></span>
					<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
				</div>
				<div class="containerBox defaultInput" style="border-bottom:none;margin-bottom:0px;padding-bottom:0px">
					<div style="display:block;margin-bottom:0px;">
						<label style="display:block;line-height:36px;font-size:12px;"><%=rb.getString("ZhiXingFangShi")%></label>
						<div id="epcStartWay" style="width:675px;height:50px;padding-top:10px;border:1px solid #DEDFE6;background:#F5F7FA;">
							<input class="alignCenter" id="viewimmediatelyWay" type="radio" disabled="disabled" name="doWay" value="immediately" style="margin-left:22px;">
							<label style="margin-right:55px;" for="viewimmediatelyWay"><%=rb.getString("LiJiZhiXing")%></label>
							<input class="alignCenter" id="viewhangUpWay" type="radio" disabled="disabled" name="doWay" value="hangUp">
							<label style="margin-right:55px;" for="viewhangUpWay"><%=rb.getString("GuaQi")%></label>
							<input class="alignCenter" id="viewsetTimeWay" type="radio" disabled="disabled" name="doWay" value="schetime">
							<label for="viewsetTimeWay"><%=rb.getString("DingShiZhiXing")%></label>
							<div style="margin-top:10px;display:inline-block;margin-left:22px;style="height:26px;width:200px;">	    
			               		<input id="viewEPCSetTime" class="easyui-datetimebox border-box border"  data-options="require:true,editable:false,readonly:true" name="scheduleStart" style="height:26px;width:200px;line-height:26px;"/>   
			                </div>
						</div>
					</div>
					
					<p style="margin-right:0px;margin-bottom:35px;">
						<label><%=rb.getString("ChiXuShiChang")%>(Min)</label>
						<input id="viewEPCDuration" type="text" name="traceIMSI" onblur="testTraceTime(this)" disabled="disabled"/>
						<span style="display:block;height:20px;line-height:20px;color:red;display:none">心灵跟踪最大时长不超过30分钟</span>
					</p>
				</div>
			</div>
		</div>
		
<div>
<script type="text/javascript">
var traceEPCJson = '${signalingTraceProperties}';

$(function(){
	var traceEPCObj = JSON.parse(traceEPCJson);

	$("#viewepcTaskTraceName").val(traceEPCObj.trace_name);
	$("#viewepcTaskTraceId").val(traceEPCObj.trace_id);
	$("#viewepcTaskRemarks").val(traceEPCObj.remarks);
	$("#viewshowEPCTraceName").val(traceEPCObj.trace_device);
	$("#viewEPCTraceIMSI").val(traceEPCObj.imsi);
	$("#viewEPCDuration").val(traceEPCObj.duration);
	if(traceEPCObj.execute_mode == 1){
		$("#viewimmediatelyWay").prop("checked","checked");
	}else if(traceEPCObj.execute_mode == 2){
		$("#viewhangUpWay").prop("checked","checked");
	}else{
		$("#viewsetTimeWay").prop("checked","checked");
		setTimeout(function(){
			$("#viewEPCSetTime").datetimebox("setValue".traceEPCObj.start_time);
		},0)
	}
	var InterfaceStr = traceEPCObj.ne_interface
	if(InterfaceStr.indexOf("HSS") == -1 && InterfaceStr.indexOf("MME") != -1){ //只有mme
		if(InterfaceStr.indexOf("S1") != -1){
			$("#viewEPCS1").prop("checked","checked");
		}
		if(InterfaceStr.indexOf("S6a") != -1){
			$("#viewEPCS6a").prop("checked","checked");
		}
	}else if(InterfaceStr.indexOf("HSS") != -1 && InterfaceStr.indexOf("MME") == -1){//只有HSS
		$("#viewHssEPCS6a").prop("checked","checked");
	}else if(InterfaceStr.indexOf("HSS") != -1 && InterfaceStr.indexOf("MME") != -1){
		$("#viewHssEPCS6a").prop("checked","checked");
		var InterfaceStrArr = InterfaceStr.split("|");
		if(InterfaceStrArr[0].indexOf("S1") != -1){
			$("#viewEPCS1").prop("checked","checked");
		}
		if(InterfaceStrArr[0].indexOf("S6a") != -1){
			$("#viewEPCS6a").prop("checked","checked");
		}
	}
})


</script>
