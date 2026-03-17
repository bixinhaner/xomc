<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#gsmBtsDetailsPage{
	height: 100%;
	width: 100%;
}
#gsmBtsDetailsPage .itemMainBoxCls{
	border-radius:10px;
	background:#fff;
	height:100%;
	width: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
}
#gsmBtsDetailsPage .itemMainBoxTitle {
	height:36px;
	padding-left: 20px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
	border-bottom: 1px solid #E9E9E9;
}
#gsmBtsDetailsPage .itemMainBoxCenter{
	width: calc(100% - 40px);
	flex:1;
	padding: 0px 20px;
	position: relative;
	overflow: auto;
}
#gsmBtsDetailsPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
	background-color: #FFFFFF;
    box-sizing: border-box;
    width: 100%;
	padding-left: 20px;
}
#gsmBtsDetailsPage .rightContentCls .contentTableTitle{
	height: 40px;
	display: flex;
	align-items: center;
	justify-content: space-between;
	font-weight: 550;
	width: 100%;
}
#gsmBtsDetailsPage .rightContentCls .contentTableTitle>div:nth-child(1){
	font-size: 12px;
}
#gsmBtsDetailsPage .moreIpItemBoxCls{
	display: flex;
	flex-wrap: wrap;
	width: 100%;
}
#gsmBtsDetailsPage .leftAndRightItemCls{
	width:40%;
	min-width:400px;
	margin-bottom: 20px;
}
#gsmBtsDetailsPage .itemListBoxCls{
	padding-top: 5px;
}
#gsmBtsDetailsPage .itemCls{
	height: 24px;
	display: inline-block;
	line-height: 24px;
	border: 1px solid #4D84FF;
	box-sizing: border-box;
	padding: 0px 10px;
	margin-right: 10px;
	margin-bottom: 10px;
}
#gsmBtsDetailsPage .itemListBoxCls .el-icon-close{
	font-size: unset;
	position: unset;
	top: unset;
	right: unset;
}
#gsmBtsDetailsPage .leftAndRightItemCls .el-input__suffix{
	height: 26px;
	display: flex;
	align-items: center;
}
#gsmBtsDetailsPage .el-form-item{
	margin-bottom: 20px;
}
#gsmBtsDetailsPage .errorBoxCls{
	color:red;
	font-size:10px;
}
#gsmBtsDetailsPage .multiPlmnEnableBoxCls .el-form-item__label{
	padding-top: 13px;
	margin-right: 20px;
}
#gsmBtsDetailsPage .el-form-item .el-form-item__label{
	font-size: 12px;
}
</style>

<div id="gsmBtsDetailsPage">
	<div class="itemMainBoxCls">
		<div class="itemMainBoxTitle">
			BTS SN: {{rowDataInfo.BTS_SerialNumber}}
			<!-- 按钮  关闭 -->
			<div class="headcloseBtn" style="top:6px;right:10px;" @click="closeBwpDetails">
				<span class="el-icon el-icon-close" style='position: unset; display: block; text-align: center;'></span>
			</div>
		</div>
		<div class="itemMainBoxCenter">
			<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="top">
				
			</el-form>
		</div>
		<div class='itemMainBoxFooter'>
			<el-button type="primary" @click="settingsSubmit"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="closeSettings" ><%=rb.getString("QuXiao")%></el-button>
		</div>
	</div>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
	regKey = /^[A-Fa-f0-9]{32}$/,
	regNumber = /^[0-9]{15}$/;
var gsmBtsDetailsPage = new Vue({
	el: '#gsmBtsDetailsPage', 
	data() {
		var vm = this,
			validateRange = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var isRequired = rule.isRequired;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;

				if(value == '' || value == undefined || value == null){
					if(isRequired){
						callback(new Error(mag))
					}else{
						callback();
					}
				}else{
					if(reg.test(value) && value >= min && value <= max){
						callback();
					}else{
						callback(new Error(mag))
					}
				}
			},
			validateFeqDomainResources = (rule,value,callback) => {
				var reg = /^[0-9]{45}$/
				if(value == '' || value == undefined || value == null){
					callback(new Error('Length<%=rb.getString("MaoHao")%> 45 Digit <%=rb.getString("ZhengXing")%>'))
				}else{
					if(reg.test(value)){
						callback();
					}else{
						callback(new Error('Length<%=rb.getString("MaoHao")%> 45 Digit <%=rb.getString("ZhengXing")%>'))
					}
				}
			};
		return {
			rowDataInfo: [],
			smallCellCode:'',
			ruleForm:{
				
			},
			rules:{
				
			},
			casts:{
				'B03E77F813ABFE7FB21855D4213678D3':'PC_CoresetZero',
				'B3E25AE53CF80DDF9B5AC8DA54F9D921':'PC_SearchSpaceZero',
				'A65CE8D56AD78CC7B7F92AACBA4AC3E0':'PC_SearchSpaceSIB1',
				'10A1CF9B17ACA6B5D8153A9383B2EB29':'PC_SearchSpaceOtherSystemInformation',
				'C6391E07216561C141EF946A4C0B285A':'PC_PagingSearchSpace',
				'3E693C4AED172F61D630176EB5514262':'PC_RaSearchSpace',

				'7CB19E81C355223B32A6BA7B9DE56EF4':'PCC_CoresetID',
				'9A3438EB70B57D9F04D92BFF735FB1BE':'PCC_FeqDomainResources',
				'86C2463E87245E3B62EC34FC295908A0':'PCC_NumSymbols',
				'6581C91FFCB40810F6AB6FC67EB7ECDF':'PCC_CceRegMappingType',
				'03B7F96860BFB3798C03662F9BD25057':'PCC_CceRegBundleSize',
				'B3FD8B3B192BDACD736FD789A83CD1B6':'PCC_CceInterleaverSize',
				'C79BBF158050C60F73194326BBA5FBF2':'PCC_CceShiftIndex',
				'9626F2D880485FD7754B759E3ECD4352':'PCC_PrecoderGranularity',

				'6575EA789C7FCF689B2DD6D6E9E186AE':'PCDList',
				'BB3B6882FC51B07BFE0F9CF7792D455A':'PCD_idx',
				'2CDE81516DAD023E8647060B88A21093':'PCD_CoresetID',
				'1732E609F87D14D007107D24EB4D09B9':'PCD_FeqDomainResources',
				'DA8FDF266453443D45F99B0B5FEF2C07':'PCD_NumSymbols',
				'3B4EC99AEDC5F91FB25E73B931F9F1D8':'PCD_CceRegMappingType',
				'3569D89C87FEDC99BE3E215EFC11C9BF':'PCD_CceRegBundleSize',
				'96988D5E3412ECAF1722387F904832EC':'PCD_CceInterleaverSize',
				'C6054DD6B2EF806020F999EA2321F30F':'PCD_CceShiftIndex',
				'E827FF7170E97CBF731B632F40804749':'PCD_PrecoderGranularity',

				'78566810F703C7C562E584E06CD56E61':'PSSCList',
				'CFD9AA6782FFA7D3AB2B0489367927B0':'PSSC_idx',
				'5535B3361C2242384C8CB3C8924E8240':'PSSC_SearchSpaceID',
				'5FFDDC889481A02359C41E3AB4FE0783':'PSSC_CoresetID',
				'C7C6EFC072DF092091CC7CFF7E730837':'PSSC_PdcchSlotPeriodicity',
				'C007391BEAB70155CF2B7312E76DC0D6':'PSSC_PdcchSlotOffset',
				'8C7E7616EE2ADE9CAF4BE25F2CF63AF9':'PSSC_SearchSpaceDuration',
				'25DDD99F18A085DD064A3EB9FF785838':'PSSC_PdcchSymbolsInSlot',
				'D34EAD2E361E39A618514FB1A3EEF53D':'PSSC_PdcchCandidatesAggLevel1',
				'0FCD14C32279F3D45D0710453788D13C':'PSSC_PdcchCandidatesAggLevel2',
				'E1059E0960A29E44ED378CB0732C4DA9':'PSSC_PdcchCandidatesAggLevel4',
				'1C693D267F7AB25E76B9D7E186AB4D82':'PSSC_PdcchCandidatesAggLevel8',
				'3E14EA8ABF208B27F19DAC2F39D5D282':'PSSC_PdcchCandidatesAggLevel16',
				'45B78476F29BD5754B02E8076638AAA6':'PSSC_SearchSpaceType',
				'B154A38869F47412665989C5D4F1FF36':'PSSC_DciFormat00And10En',

				'91A4FD63FF109B64FABCEAF46863A94B':'PDSCHC_DmrsAdditionalPosition',
				'EC384CE6F251888001C5F2C2D07D65E0':'PDSCHC_MaxMimo',
				'478F544D27BAF2B92A8687A478A9D8F3':'PDSCHC_McsTable',

				'5AE506BD39C41973416894D7280B8F4A':'PDSCHSSCList',
				'AA3AA8A643272520C1F96F1CD9AE502B':'PDSCHSSC_idx',
				'B9952A5F420C38D3B43EB280D664ACC9':'PDSCHSSC_StartSymbolAndLength',

				'27418176D4F9B87E80CE3FA012579841':'PDSCHDList',
				'BA1A172362789DAB986C4AC7F5AED02A':'PDSCHD_idx',
				'3775A5CA31ACEB996D605EF13C4CA2F4':'PDSCHD_StartSymbolAndLength',
			},
			codeList:[],
			optType:'',
			tbType:'',
			
		};
	},
	computed: {
		
	},
	methods: {
		init(row,code){
			var vm = this;
			vm.rowDataInfo = row;
			vm.smallCellCode = code;
			var codeList=[];
			Object.keys(vm.casts).forEach(function(key){
				codeList.push(key)
			});
			vm.codeList = codeList;
			vm.getParamData(vm.smallCellCode,'23003');
		},
		getParamData(code,id) {
			var vm = this,
				codes = [],
				url = '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
				params = {
					id: id,
					btsIndex:vm.rowDataInfo.BTS_idx,
                    trxChannelIndex:vm.rowDataInfo.BTS_NumOfTrxChannel,
					smallCellCode: code
				};
			axios.post(url, stringify(params)).then(function(res){
				var data = res.data;
				vm.resetFormData();
				if(data && Array.isArray(data)) {
					data.map(function(item){
						item.groups.map(function(group){
							group.list.map(function(m){
								if(m.type == 'list'){
									vm.initTable(m.url,m.label);
								}else{
									codes.push(m.name);
									// 执行赋值
									vm.setValue(m);
								}
							});
						});
					});
					initForm(vm.$refs.ruleForm);
				}
			});
		},
		// 映射赋值
		setValue(item) {
			var vm = this,
			code = item.name,
			value = item.value;

			// indexs是否含有
			var key = vm.casts[code];
			try{
				if(key){
					vm.ruleForm[key] = value;
				}
			}catch(e){}
		},
		initTable(url,type){
			var vm = this,codes = [];
			var params = {
					smallCellCode : vm.smallCellCode,
					btsIndex:vm.rowDataInfo.BTS_idx,
                    trxChannelIndex:vm.rowDataInfo.BTS_NumOfTrxChannel,
			}
			axios.post(url,stringify(params)).then(res=>{
				var data = res.data;
				if(data.rows){
					data.rows.map(item=>{
						var obj = {};
						for(var key in item){
							codes.push(key);
							obj[vm.casts[key]] = item[key]
						}
						
					})
					initForm(vm.$refs.ruleForm);
				}
			})
		},
		// 重置form数据
		resetFormData(){
			var vm =this;
				params={
					
				};
			Object.assign(vm.ruleForm,params);
		},
		// 判断是否为空
		isNull(val){
			if(val==undefined || val == null || val =="") return true;
			else return false;
		},
		// 验证输入的是否是整数
		isInteger(str) {
			if(str.length==0){
				return false;
			}
			var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
			if(!reg.test(str)){
				return false;
			}
			return true;  
		},
		tableRowClassName({row,rowIndex}){
			if(row.operateType && row.operateType == 'remove'){
				return 'hidden-row'
			}
			return ''
		},
		
		settingsSubmit(){
			var vm = this;
			var params = {},
				isChanged = isFormChanged(vm.$refs.ruleForm),
				isSync = false;

			if(!isChanged){
				showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
				return;
			}

			vm.$refs.ruleForm.fields.map(function(field){
				var key = vm.getNameByProp(field.prop);

				if(Array.isArray(field.fieldValue)){
					var vList = field.fieldValue.map(function(item){return item}),
						oList = (field.reinitialValue||[]).map(function(item){return item}),
						val = JSON.stringify(vList.sort()),
						orVal = JSON.stringify(oList.sort());

					if(val != orVal) {
						var editList=[],subList=[];
						vList.map((items)=>{
							if(items.operateType){
								if(field.prop == 'PDSCHSSCList'){
									var StartSymbol = items['PDSCHSSC_StartSymbol'] ? parseInt(items['PDSCHSSC_StartSymbol']) : '',
										Length = items['PDSCHSSC_Length'] ? parseInt(items['PDSCHSSC_Length']) : '';
									if((Length - 1) <= 7){
										items['PDSCHSSC_StartSymbolAndLength'] = 14*(Length - 1) + StartSymbol + '';
									}else{
										items['PDSCHSSC_StartSymbolAndLength'] = 14*( 14 - Length + 1) + ( 14 - 1 - StartSymbol) + '';
									}
								}else if(field.prop == 'PDSCHDList'){
									var StartSymbol = items['PDSCHD_StartSymbol'] ? parseInt(items['PDSCHD_StartSymbol']) : '',
										Length = items['PDSCHD_Length'] ? parseInt(items['PDSCHD_Length']) : '';
									if((Length - 1) <= 7){
										items['PDSCHD_StartSymbolAndLength'] = 14*(Length - 1) + StartSymbol + '';
									}else{
										items['PDSCHD_StartSymbolAndLength'] = 14*( 14 - Length + 1) + ( 14 - 1 - StartSymbol) + '';
									}
								}
								editList.push(items)
							}
						})
						editList.map((items)=>{
							if(items.operateType == 'add'){
								Object.keys(items).map((key)=>{
									if(key.slice(-3) == 'idx'){
										delete items[key]
									}
								})
							}
						})
						
						editList.map((items)=>{
							var objs={};
							for(var listVal in items){
								var listKey = vm.getNameByProp(listVal);
								if(listVal != 'Xn_Status' && listVal != 'PDSCHSSC_StartSymbol' && listVal != 'PDSCHSSC_Length' && listVal != 'PDSCHD_StartSymbol' && listVal != 'PDSCHD_Length'){
									objs[listKey] = items[listVal]
								}
							}
							objs.cellIndex = '1';
							objs.bwpIndex = vm.rowDataInfo.DLBWP_idx;
							subList.push(objs)
						})
						params[key] = subList;
					};
				}else{
					if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
						
					}else if(field.fieldValue != field.reinitialValue) {
						var editData={
							cellIndex:vm.cellIndex,
							bwpIndex:vm.rowDataInfo.DLBWP_idx,
							value:field.fieldValue
						}
						params[key] = editData;

					};
				}
			});
			
			vm.$refs.ruleForm.validate(function(valid){
				if(valid) {
					var rowCode = vm.smallCellCode,
						url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;
					$('#gsm_setting_main').addClass('loading');
					axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
							vm.closeSettings();
						}else{
							vm.$message.error(data["message"])
						}
						$('#gsm_setting_main').removeClass('loading');
					})
				}
			});
		},
		getNameByProp(prop) {
			var vm = this,
				reg = /^\w*\.\d*\.\w*$/
				key = prop;
			
			if(reg.test(prop)) {
				var mReg = /\.(\d*)\./,
					sufReg = /\.(\w*)$/,
					idx = prop.match(mReg)[1],
					sufStr = prop.match(sufReg)[1];

				vm.codeList.map(function(name){
					var index = vm.indexs[name];
					if(vm.casts[name] == sufStr && index == idx) {
						key = name;
					}
				});
			}else {
				vm.codeList.map(function(name){
					if(vm.casts[name] == prop) {
						key = name;
					}
				});
			}

			return key;
		},
		createId(idVal,list){
			var vm = this,
				val = idVal + '';
			if(list.includes(val) == true){
				idVal += 1 ;
				return vm.createId(idVal,list);
			}else{
				return  idVal + '';
			}
		},
		closeSettings(){
            eventBus.$emit('gsm-close-setting');
		},
		//校验IP
        isValidIP(ip){
            var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
            return reg.test(ip);     
        },
        //Ipv6校验 
        isIPv6(str){ 
            var reg = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}$|^:((:[\da-fA-F]{1,4}){1,6}|:)$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?$|^([\da-fA-F]{1,4}:){6}:$/
            return reg.test(str);
        },
        //校验子网掩码
        isMask(str){
            var exp=/^(254|252|248|240|224|192|128|0)\.0\.0\.0|255\.(254|252|248|240|224|192|128|0)\.0\.0|255\.255\.(254|252|248|240|224|192|128|0)\.0|255\.255\.255\.(254|252|248|240|224|192|128|0)$/; 
            return exp.test(str); 		
        },
        // 验证输入的是否是数字
        isNumeric(str) {
            if(str.length==0){
                return false;
            }
            for(var i=0;i<str.length;i++){
                if(str.charAt(i)<"0" || str.charAt(i)>"9"){
                    return false;
                }
            }
            return true;  
        },
		syncSettingsClick(){
			var vm = this,
				urls='${ctx}/cell/quicksettings/sync.action',
				params = {
					smallCellCode:vm.smallCellCode
				},
				str = Math.random().toString();
				
			axios.post(urls,stringify(params)).then(res=>{
				var data = res.data;
				if(data["success"]){
					gnbTabSettingVue.changeMain('coreNetwork');
				}else{
					vm.$message.error(data["message"])
				}
			})
		},
		closeBwpDetails(){
			gsmSettingBtsPageVue.$refs.sharingSlide.hide();
		},
		PDSCHSSC_StartSymbolAndLengthChange(val){
			var vm = this;
			var StartSymbol = vm.addPDSCHSSCDialogForm['PDSCHSSC_StartSymbol'];
			var Length = vm.addPDSCHSSCDialogForm['PDSCHSSC_Length'];
			StartSymbol = parseInt(StartSymbol);
			Length = parseInt(Length);
			if(vm.isNumeric(StartSymbol) && vm.isNumeric(Length)){
				if((Length - 1) <= 7){
					vm.addPDSCHSSCDialogForm['PDSCHSSC_StartSymbolAndLength'] = (14*(Length - 1) + StartSymbol) + '';
				}else{
					vm.addPDSCHSSCDialogForm['PDSCHSSC_StartSymbolAndLength'] = 14*( 14 - Length + 1) + ( 14 - 1 - StartSymbol) + '';
				}
			}else{
				vm.addPDSCHSSCDialogForm['PDSCHSSC_StartSymbolAndLength'] = '';
			}
		},
		PDSCHD_StartSymbolAndLengthChange(val){
			var vm = this;
			var StartSymbol = vm.addPDSCHDDialogForm['PDSCHD_StartSymbol'];
			var Length = vm.addPDSCHDDialogForm['PDSCHD_Length'];
			StartSymbol = parseInt(StartSymbol);
			Length = parseInt(Length);
			if(vm.isNumeric(StartSymbol) && vm.isNumeric(Length)){
				if((Length - 1) <= 7){
					vm.addPDSCHDDialogForm['PDSCHD_StartSymbolAndLength'] = (14*(Length - 1) + StartSymbol) + '';
				}else{
					vm.addPDSCHDDialogForm['PDSCHD_StartSymbolAndLength'] = 14*( 14 - Length + 1) + ( 14 - 1 - StartSymbol) + '';
				}
			}else{
				vm.addPDSCHDDialogForm['PDSCHD_StartSymbolAndLength'] = '';
			}
		}
	},
	mounted() {
		eventBus.$off("btsDetails-init").$on("btsDetails-init",this.init)
	}
});

</script>
