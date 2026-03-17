<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#alarmNoticePage .headerBoxCls{
		height: 50px;
		width: 100%;
		position: relative;
		border-bottom: 1px solid #E9E9E9;
		display: flex;
		align-items: center;
		font-size: 16px;
		font-weight: bold;
		padding-left: 20px;
	}
	#alarmNoticePage .emailGroupBox {
		display: flex;
		margin: 20px 40px;
		flex-direction: column;
	}
	#alarmNoticePage .emailGroupBox > div{
		margin-bottom: 20px;
		padding-left: 25px;
	}
	#alarmNoticePage .headerTitleBox span:nth-child(2){
		font-size: 14px;
		color: #666666;
		font-weight: bold;
		margin-left:5px;
	}
	.emailPromptCls{
		font-size: 12px;
		color: #BBBBBB;
		position: relative;
		top: 0px;
	}
	.emailPromptCls .el-icon:before{
		color: #BBBBBB;
		font-size: 12px;
	}
	#alarmNoticePage .errorBoxCls{
		color: #FA5555;
	}
	.alarmBottomLine{
		background-color:#E9E9E9;
		width: calc(100% + 30px);
		height: 1px;
		margin-left: -30px;
	}
	#alarmNoticePage .customSettingBox{
		margin: 20px 40px;
	}
	#alarmNoticePage .customNoticeTableBox{
		height: 350px;
		width: 60%;
		border: 1px solid #E9E9E9;
	}
	#customNoticeModifyForm .el-form-item{
		margin-bottom: 20px;
	}
	#alarmNoticePage .emailEnableClickBox{
		height: 20px;
		width: 100%;
		position: absolute;
		top:0px;
		left: 0px;
		right: 0px;
		bottom: 0px;
		margin: auto;
		z-index: 66;
		cursor: pointer;
		opacity: 0;
	}
	.tooltipCls.is-dark{
		background : #959595 ;
		color : #FFFFFF ;
	}
	.tooltipCls[x-placement^=top] .popper__arrow ,
	.tooltipCls[x-placement^=top] .popper__arrow::after{
		border-top-color: #959595!important;
	}

	.tooltipCls[x-placement^=bottom] .popper__arrow ,
	.tooltipCls[x-placement^=bottom] .popper__arrow::after {
		border-bottom-color: #959595!important;
	}
	.tooltipCls[x-placement^=right] .popper__arrow ,
	.tooltipCls[x-placement^=right] .popper__arrow::after {
		border-right-color: #959595!important;
	}
	.tooltipCls[x-placement^=left] .popper__arrow ,
	.tooltipCls[x-placement^=left] .popper__arrow::after {
		border-left-color: #959595!important;
	}
	.blueIcon::before{
		font-size: 14px!important;
		color: #4D84FF;
	}
	.whiteIcon::before{
		font-size: 14px!important;
		color: #FFFFFF;
	}
	.grayIcon::before{
		color: #B8C3D9;
		font-size: 16px!important;
	}
	.greenIcon::before{
		color: #67D972;
		font-size: 16px!important;
	}
</style>
<div id='alarmNoticePage' style="position: relative;overflow:hidden">
	<div style="height:100% ;overflow:auto">
		<div class="splitGroup">
			<div class="splitGroup_title">
				<span><%=rb.getString("GaoJingTiShiYin")%></span>
			</div>
			<div class="splitGroup_body">
				<div style="margin-bottom: 20px;">
					<span><%=rb.getString("GaoJingTiShiKaiGuan")%></span>
					<el-switch 
						v-model="ruleForm.voiceEnable" 
						active-value="1" 
						inactive-value="0"
						style="margin-left:15px;"
						:disabled="isReadonly"
						@change="voiceChange">
					</el-switch>
				</div>
				<div>
					<span><%=rb.getString("GaoJingJiBie")%></span>
					<el-checkbox-group v-model="ruleForm.voiceSeverity" style="margin-left:15px;display:inline-block;" :disabled="isReadonly" @change="voiceChange">
						<el-checkbox label="31001" ><%=rb.getString("JinJiGaoJing")%></el-checkbox>
						<el-checkbox label="31002"><%=rb.getString("ZhuYaoGaoJing")%></el-checkbox>
						<el-checkbox label="31003" ><%=rb.getString("CiYaoGaoJing")%></el-checkbox>
						<el-checkbox label="31004" ><%=rb.getString("JingGaoGaoJing")%></el-checkbox>
					</el-checkbox-group>
				</div>
			</div>
		</div>
		<!-- 邮件通知分组 -->
		<div class="splitGroup">
			<div class="splitGroup_title">
				<span><%=rb.getString("XinJianGaoJingTongZhi")%></span>
			</div>
			<div class="splitGroup_body" >
				<!-- 邮件通知开关 -->
				<div>
					<span><%=rb.getString("XinJianGaoJingTongZhi")%></span>
					<el-switch 
						v-model="ruleForm.enable"
						:before-change="globalEmailEnableClick"
						active-value="1" 
						inactive-value="0"
						style="margin-left:15px;"
						:disabled="isReadonly"
					></el-switch>
					<div class="emailPromptCls" style="display:inline-block;margin-left:10px;">
						<span v-show="globalEmailEnableTiShi == '2' || globalEmailEnableTiShi == '3'" class="el-icon-circle-info el-icon" style="margin-right:5px;" ></span>
						<span v-show="globalEmailEnableTiShi == '2'"><%=rb.getString("GuanLiYuanYouXiangSheZhi")%></span>
						<span v-show="globalEmailEnableTiShi == '3'"><%=rb.getString("PuTongYongHuYouXiangSheZhi")%></span>
					</div>
				</div>
				<!-- 默认邮箱输入 -->
				<div>
					<div style="margin:10px 0;"><%=rb.getString("YouXiang")%></div>
					<el-input 
						type="textarea" 
						resize="none" 
						:rows="3" 
						maxlength="500" 
						v-model="ruleForm.emailAddress" 
						style="width: 580px;"
						@blur="globalEmailBlur"
						:disabled="isReadonly">
					</el-input>
					<div v-if="!isReadonly" style="display:inline-block;">
						<el-button type="primary" @click="settingGlobalSubmit"><%=rb.getString("QueDing")%></el-button>
						<el-button @click="settingGlobalClose"><%=rb.getString("QuXiao")%></el-button>
					</div>
					<div class="emailPromptCls"><span class="el-icon-circle-info el-icon" style="margin-right:5px;" ></span><%=rb.getString("YouXiangDiZhiTiShi")%></div>
					<div class="errorBoxCls" v-if="globalEmailErrorShow">{{ruleForm.errorMessage}}</div>
				</div>

				<!-- 告警模板邮件通知设置 -->
				<div style="margin:10px 0;">
					<span><%=rb.getString("ZiDingYiSheZhi")%></span>
				</div>
				<div class="customNoticeTableBox">
					<el-ctable 
						id="customNoticeTable" 
						:url="customNoticeTableUrl"
						ref="customNoticeTable" 
						:height="height"
						:row-key="'templateId'"
						@row-click="customNoticeTableRowClick"
						:page-size="pageSize" pagination="true"
						@selection-change='customNoticeSelect'>

						<el-table-column type='selection' width="45" :reserve-selection="true" v-if="!isReadonly"></el-table-column>
						<el-table-column label='' width="50" prop="" v-if="!isReadonly">
							<template slot-scope="scope">
								<el-tooltip popper-class="tooltipCls" content='<%=rb.getString("XiuGai")%>' placement='bottom'>
									<span class="el-icon el-icon-operation-edit" @click="customNoticeModifyClick(scope.row,event)" style="cursor: pointer;"></span>
								</el-tooltip>
							</template>
						</el-table-column>
						<el-table-column prop="emailEnable" width="100" label="<%=rb.getString("SheZhiKaiGuan")%>">
							<template slot-scope="scope">
								<div style="position:relative;">
									<el-switch
										v-model="scope.row.emailEnable"
										:before-change="emailEnableClick"
										active-color="#4D84FF" 
										inactive-color="#CFCFCF" 
										:active-value="1" 
										:inactive-value="0"
										:disabled="isReadonly"
									></el-switch>
								</div>
							</template>
						</el-table-column>
						<el-table-column prop="templateName" label="<%=rb.getString("MuBanMingCheng")%>"></el-table-column>
						
						<el-table-column prop="emailPeriod" label="<%=rb.getString("XinJianGaoJingShiJianJianGe")%>" :formatter="emailPeriodFmt"></el-table-column>
						<el-table-column prop="emailAddress" label="<%=rb.getString("YouXiang")%>">
							<template slot-scope="scope">
								<div v-if="!['','NULL','null',null,undefined].includes(scope.row.emailAddress)" style="display: flex;align-items: center;">
									<span v-html="emailAddressFmt(scope.row.emailAddress)" ></span>
									<el-popover trigger="click" v-if="parseEmailAddress(scope.row.emailAddress).length !== 1">
										<span v-if="parseEmailAddress(scope.row.emailAddress).length !== 1" style="color:#4d84ff;cursor: pointer;" slot="reference">
											(<span v-html="parseEmailAddress(scope.row.emailAddress).length"></span>)
										</span>
										<div style="padding:10px 10px 0px 10px;">
											<div v-for="item in parseEmailAddress(scope.row.emailAddress)" style="margin-bottom:10px;">
												{{item}}
											</div>
										</div>
									</el-popover>
								</div>
								<div></div>
							</template>
						</el-table-column>
						<el-table-column prop="globalEmailEnable" label="<%=rb.getString("TongZhiYouJianZu")%>" >
							<template slot-scope="scope">
								<div v-if="scope.row.globalEmailEnable == '1'" style="display:flex;align-items:center;">
									Yes
								</div>
								<div v-if="scope.row.globalEmailEnable == '0'">
									No
								</div>
							</template>
						</el-table-column>
					</el-ctable>
				</div>
			</div>
		</div>
	</div>
	<!-- 批量操作  -->
	<el-bulk target="customNoticeTable" :list="customNoticeSelection" :row-key="'templateId'" show-prop="templateName"
		:message="bulkTableMessage">
		<template slot="button">
			<a class="linkbutton" style="margin-right:-4px;" @click="useGlobalClick('','1','batch')">
				<!--<span class="el-icon el-icon-operation-enable1 whiteIcon" style="margin-right:5px;"></span>-->
				<%=rb.getString("TongZhiYouJianZu")%>
			</a>
			<a class="linkbutton linkbutton_nowanna" @click="useGlobalClick('','0','batch')">
				<!--<span class="el-icon el-icon-operation-disable1 blueIcon" style="margin-right:5px;"></span>-->
				<%=rb.getString("BuTongZhiYouJianZu")%>
			</a>
		</template>
	</el-bulk>
	<el-dialog :title='dialogTitle' id="customNoticeDialog" :visible.sync="showCustomNoticeDialog" top="15vh" ref="customNoticeDialog" width="780px" 
		:close-on-click-modal="false"  @close='customNoticeModifyClose' append-to-body>
		<el-form  :model="customNoticeModifyForm" ref="customNoticeModifyForm" :rules="customNoticeModifyRules" label-position="top" id="customNoticeModifyForm">
			<el-form-item prop='templateName' label="" label-width="160px">
				<span style="display:inline-block;width:110px;"><%=rb.getString("MuBanMingCheng") %></span>
				<span>{{customNoticeModifyForm.templateName}}</span>
			</el-form-item>
			<el-form-item prop='emailEnable' label="" label-width="160px">
				<span style="display:inline-block;width:110px;"><%=rb.getString("SheZhiKaiGuan") %></span>
				<el-switch 
					v-model="customNoticeModifyForm.emailEnable"
					@change="emailEnableChange"
					active-color="#4D84FF" 
					inactive-color="#CFCFCF" 
					:active-value="1" 
					:inactive-value="0" 
				></el-switch>
				<div class="emailPromptCls" style="display:inline-block;margin-left:10px;">
					<span v-show="emailEnableTiShi == '2' || emailEnableTiShi == '3'" class="el-icon-circle-info el-icon" style="margin-right:5px;" ></span>
					<span v-show="emailEnableTiShi == '2'"><%=rb.getString("GuanLiYuanYouXiangSheZhi")%></span>
					<span v-show="emailEnableTiShi == '3'"><%=rb.getString("PuTongYongHuYouXiangSheZhi")%></span>
				</div>
			</el-form-item>
			<el-form-item prop='emailPeriod' label="<%=rb.getString("XinJianGaoJingShiJianJianGe") %>" label-width="160px">
				<el-select v-model="customNoticeModifyForm.emailPeriod">
					<el-option label="Real Time" :value="5"></el-option>
					<el-option label="10 Minute" :value="10"></el-option>
					<el-option label="30 Minute" :value="30"></el-option>
					<el-option label="60 Minute" :value="60"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item prop='remainTime' label="<%=rb.getString("RongRenShiChang") %>" label-width="160px">
				<el-select v-model="customNoticeModifyForm.remainTime">
					<el-option label="Real Time" :value="5"></el-option>
					<el-option label="10 Minute" :value="10"></el-option>
					<el-option label="30 Minute" :value="30"></el-option>
					<el-option label="60 Minute" :value="60"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item prop='emailAddress' label="<%=rb.getString("YouXiang")%>" label-width="160px">
				<el-input type="textarea" resize="none" :rows="3" maxlength="500" v-model="customNoticeModifyForm.emailAddress" style="width: 500px;"></el-input>
				<div class="emailPromptCls"><span class="el-icon-circle-info el-icon" style="margin-right:5px;" ></span><%=rb.getString("YouXiangDiZhiTiShi")%></div>
			</el-form-item>
			<el-form-item prop='globalEmailEnable' label="" label-width="160px">
				<el-checkbox v-model="customNoticeModifyForm.globalEmailEnable" true-label="1" false-label="0"><%=rb.getString("TongZhiYouJianZu")%></el-checkbox>
			</el-form-item>
		</el-form>
		<span slot="footer">
			<div>
				<el-button type="primary" @click="customNoticeModifySubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="customNoticeModifyClose"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</span>
	</el-dialog>
</div>
<script type="text/javascript">
var operatorCode = operatorCodeGloab;
var emailReg = /^(([a-zA-Z0-9_\.-]+)@([\da-z\.-]+)\.([a-z\.]{2,6}\;))*([a-zA-Z0-9_\.-]+)@([\da-z\.-]+)\.([a-z\.]{2,6})$/;
new Vue({
	el:'#alarmNoticePage',
	data(){
		// 邮箱验证
		var validatorEmail = (rule,value,callback) => {
			
				if(value){
					if(emailReg.test(value)){
						callback()
					}else{
						callback(new Error('<%=rb.getString("YouXiangGeShiCuoWu")%>'))
					}
				}else{
					callback()
				}
			
		};
		return {
			ruleForm:{
				voiceEnable:'0',
				voiceSeverity:['31001','31002'],
				enable:'1',
				emailAddress:'',
				errorMessage:''
			},
			oldForm:{
				voiceEnable:'0',
				voiceSeverity:['31001','31002'],
			},
			customNoticeTableUrl:'${ctx}/fault/viewConfig/queryGlobalConfigurationTemplateInfo.action?operatorCode='+operatorCode,
			height:'100%',
			pageSize:50,
			dialogTitle:'',
			showCustomNoticeDialog:false,
			customNoticeModifyForm:{
				templateName:'',
				emailEnable:0,
				emailPeriod:10,
				remainTime:10,
				emailAddress:'',
				globalEmailEnable:'0'
			},
			customNoticeModifyRules:{
				emailAddress:[
					{validator:validatorEmail,trigger:'blur'}
				],
			},
			globalEmailEnableTiShi:'1',
			emailEnableTiShi:'1',
			customNoticeSelection:[],
			bulkTableMessage:{title:'<%=rb.getString("YiXuan")%>',subTitle:'<%=rb.getString("MuBanMingCheng")%>',clear:'<%=rb.getString("QingKong")%>',cancel:'<%=rb.getString("QuXiao")%>'},
			customNoticeTableRow:{},
		}
	},
	computed: {
		globalEmailErrorShow(){
			return this.ruleForm.emailAddress ? true : false;
		},
		isReadonly(){
			return writableMap['CODE_ALARM_VIEW'] == false;
		}
	},
	methods:{
		// 初始化请求数据
		init(){
			var vm = this,
				params = {
					operatorCode:operatorCode
				};
			axios.post('${ctx}/fault/viewConfig/queryGlobalEnable.action',stringify(params)).then(function(response){
				var data = response.data
				if(data){
					vm.ruleForm.enable = data.enable ? data.enable : '0';
					vm.ruleForm.emailAddress = data.emailAddress ? data.emailAddress : '';
				}
			})

			axios.post('${ctx}/fault/viewConfig/queryAlarmAlertInfo.action',stringify(params)).then(function(response){
				var data = response.data
				if(data){

					vm.ruleForm.voiceEnable = data.enable + '';
					vm.ruleForm.voiceSeverity = data.severityType ? data.severityType.split(',') : [];

					vm.oldForm.voiceEnable = data.enable + '';
					vm.oldForm.voiceSeverity = data.severityType ? data.severityType.split(',') : [];
				}
			})
			
		},
		voiceChange(){
			var vm = this,params={};

			var arr = vm.ruleForm.voiceSeverity;
			params.servityType = arr.join(',');
			params.enable = vm.ruleForm.voiceEnable;

			axios.post('${ctx}/fault/viewConfig/saveAlarmAlertInfo.action',stringify(params)).then(function(response){
				var data = response.data;
				if(data.success){
					vm.$message({
						message: '<%=rb.getString("ChengGong")%>',
						type: 'success'
					});
				}else{
					vm.$message({
						message: data.message,
						type: 'error'
					});
					vm.ruleForm.voiceEnable = vm.oldForm.voiceEnable;
					vm.ruleForm.voiceSeverity = vm.oldForm.voiceSeverity;
				}
			})
		},
		// 告警通知设置 关闭
		closeNoticePage(){
			var vm = this;
			alarmViewVue.$refs.sharingSlide.hide();
		},
		// 全局告警通知按钮事件
		globalEmailEnableClick(fn){
			var vm = this,
				params = {
					operatorCode:operatorCode,
					emailAddress:vm.ruleForm.emailAddress,
					enable:vm.ruleForm.enable == '1' ? '0' : '1'
				};
			if(params.enable == '1'){
				//判断当前是否填写 setting 中的email 判断是否可以新建
				axios.post("${ctx}/cell/fault/queryHasSettingEmailConfig.action").then(function(response){
					if(response.data.result == '2'){
						vm.globalEmailEnableTiShi = '2';
					}
					if(response.data.result == '3'){
						vm.globalEmailEnableTiShi = '3';
					}
				})
			}else{
				vm.globalEmailEnableTiShi = '1';
			}
			axios.post('${ctx}/fault/viewConfig/saveGlobalEnableInfo.action',stringify(params)).then(function(response){
				var data = response.data
				if(data.success){
					fn();
					vm.$message({
						message: '<%=rb.getString("ChengGong")%>',
						type:'success',
					});
					vm.init();
				}else{
					vm.$message({
						message: data.message,
						type:'error',
					});
				}
			})

		},
		// 邮件组输入域失焦事件
		globalEmailBlur(){
			var vm = this;
			if(vm.ruleForm.emailAddress){
				if(emailReg.test(vm.ruleForm.emailAddress)){
					vm.ruleForm.errorMessage = '';
				}else{
					vm.ruleForm.errorMessage = '<%=rb.getString("YouXiangGeShiCuoWu")%>'
					return
				}
			}else{
				vm.ruleForm.errorMessage = '';
			}
		},
		//  邮件组设置 提交
		settingGlobalSubmit(){
			var vm = this,
				urls = '${ctx}/fault/viewConfig/saveGlobalEmailAddressInfo.action',
				params = {
					operatorCode:operatorCode,
					emailAddress:vm.ruleForm.emailAddress,
					enable:vm.ruleForm.enable == '1' ? '0' : '1'
				};
			
			if(vm.ruleForm.emailAddress){
				if(emailReg.test(vm.ruleForm.emailAddress)){
					vm.ruleForm.errorMessage = '';
				}else{
					vm.ruleForm.errorMessage = '<%=rb.getString("YouXiangGeShiCuoWu")%>'
					return
				}
			}else{
				vm.ruleForm.errorMessage = '';
			}
			axios.post(urls,stringify(params)).then(function(response){
				var data = response.data;
				if(data.success){
					vm.$message({
						message: '<%=rb.getString("ChengGong")%>',
						type:'success',
					});
				}else{
					vm.$message({
						message: data.message,
						type:'error',
					});
				}
			})
					
		},
		// 邮件组设置 取消
		settingGlobalClose(){
			var vm = this;
			vm.init();
		},
		// 打开自定义修改弹窗
		customNoticeModifyClick(row){
			var vm = this;
			vm.dialogTitle = '<%=rb.getString("XiuGai")%>';
			Object.assign(vm.customNoticeModifyForm,row)
			vm.showCustomNoticeDialog = true;
		},
		// 自定义修改 提交
		customNoticeModifySubmit(){
			var vm = this,
				urls = '${ctx}/fault/viewConfig/saveGlobalEmailConfigInfo.action',
				params = {
					operatorCode:operatorCode,
					templateId:vm.customNoticeModifyForm.templateId,
					templateName:vm.customNoticeModifyForm.templateName,
					emailEnable:vm.customNoticeModifyForm.emailEnable,
					emailAddress:vm.customNoticeModifyForm.emailAddress,
					emailPeriod:vm.customNoticeModifyForm.emailPeriod,
					remainTime:vm.customNoticeModifyForm.remainTime,
					globalEmailEnable:vm.customNoticeModifyForm.globalEmailEnable,
				};
			vm.$refs["customNoticeModifyForm"].validate( valid => {
				if(valid){
					axios.post(urls,stringify(params)).then(function(response){
						var data = response.data;
						if(data.success){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							});
							vm.showCustomNoticeDialog = false;
							vm.$refs.customNoticeTable.refresh();
						}else{
							vm.$message({
								message: data.message,
								type:'error',
							});
						}
					})
				}
			}) 
			
		},
		// 自定义修改 取消
		customNoticeModifyClose(){
			var vm = this,
				params={
					templateName:'',
					emailEnable:0,
					emailPeriod:0,
					emailAddress:'',
					globalEmailEnable:'1'
				};
			vm.showCustomNoticeDialog = false;
			Object.assign(vm.customNoticeModifyForm,params)

		},
		
		// 告警通知按钮改变事件
		emailEnableChange(val){
			var vm = this;
			if(val == 1){
				//判断当前是否填写 setting 中的email 判断是否可以新建
				axios.post("${ctx}/cell/fault/queryHasSettingEmailConfig.action").then(function(response){
					if(response.data.result == '2'){
						vm.emailEnableTiShi = '2';
					}else if(response.data.result == '3'){
						vm.emailEnableTiShi = '3';
					}
				})
			}else{
				vm.emailEnableTiShi = '1';
				vm.$refs.customNoticeModifyForm.clearValidate('emailAddress');
			}
		},
		/**
		 * 全局设置 启用、禁用事件
		 * type: single 单个  batch 批量
		**/ 
		useGlobalClick(row,value,type){
			var vm = this,
				urls = '${ctx}/fault/viewConfig/applyGlobalEmailConfig.action',
				params={
					operatorCode:operatorCode,
					templateIds:'',
					globalEmailEnable:value
				},
				dataList = [];
			if(type == 'single'){
				params.templateIds = row.templateId;
			}else{
				vm.customNoticeSelection.map((item)=>{
					dataList.push(item.templateId)
				})
				params.templateIds = dataList.join(',')
			}
			axios.post(urls,stringify(params)).then(function(response){
				var data = response.data;
				if(data.success){
					vm.$message({
						message: '<%=rb.getString("ChengGong")%>',
						type:'success',
					});
					if(type == 'batch'){
						vm.$refs.customNoticeTable.clearSelection();
					}
					vm.$refs.customNoticeTable.refresh();
				}else{
					vm.$message({
						message: data.message,
						type:'error',
					});
				}
			})
		},
		// 告警通知模板 表格行点击事件
		customNoticeTableRowClick(row,column,event){
			var vm = this;
			vm.customNoticeTableRow = row;
		},
		// 告警通知 开启、关闭事件
		emailEnableClick(fn){
			var vm = this,
				urls = '${ctx}/fault/viewConfig/applyEmailEnableConfig.action';
			vm.$nextTick(()=>{
				var row = vm.customNoticeTableRow,
					params={
						operatorCode:operatorCode,
						templateIds:row.templateId,
						emailPeriod:row.emailPeriod,
						emailEnable:''
					};
				if(row.emailEnable === 0){
					params.emailEnable = '1';
				}else{
					params.emailEnable = '0';
				}
				axios.post(urls,stringify(params)).then(function(response){
					var data = response.data;
					if(data.success){
						fn();
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success',
						});
						vm.$refs.customNoticeTable.refresh();
					}else{
						vm.$message({
							message: data.message,
							type:'error',
						});
					}
				})
			
			});
		},
		// 不开启的模板不可选
		checkSelectTable(row,index){
			return row.emailEnable !== 0;
		},
		// 认证用户表格选择
		customNoticeSelect(selection){
			var vm = this;
			vm.customNoticeSelection = selection
		},
		//字段格式化
		emailPeriodFmt(row,column,value,rowIndex){
			var code ={
				5:'Real Time',
				10:'10 Minute',
				30:'30 Minute',
				60:'60 Minute'
			}
			return code[value]
		},
		emailAddressFmt(value){
			var emailArr = value.split(';');
			return emailArr[0]
		},
		parseEmailAddress(value){
			var emailArr = value.split(';');
			return emailArr
		}
	},
	mounted(){
		this.init();
	}
	
})

</script> 