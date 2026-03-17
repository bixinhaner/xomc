<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<div id='enbSettingNewPanel' style='display:flex;width:100%;height:100%;'>
	<div style='min-width:115px;border-right:1px solid #EEE;overflow:hidden'>
		<ul class='left-title-list'>
			<li v-for="item in menuList" v-show="item.show" :class="{active:activeCode == item.code, disabled: item.disabled}" @click="changeMenu(item.code,item.id, item.disabled)">{{item.text}}</li>
		</ul>
	</div>
	<div style='flex:auto;overflow:auto' id="enbSettingRightPage"></div>
</div>
<script>
	var enbSettingNewPanelVue = new Vue({
		el:'#enbSettingNewPanel',
		data(){
			return{
				menuList:[],
				activeCode:'Network',
				smallCellCode:'',
				sn: '',
				connStatus: ''
			}
		},
		methods:{
			init(code, sn, connStatus, carrierType){
				var vm = this;
				vm.smallCellCode = code;
				vm.sn = sn;
				vm.connStatus = connStatus;
				var params = {
						title:'Settings',
						smallCellCode:code
					},
					disabled = connStatus == 'Off' || carrierType == '2';
				
				axios.post("${ctx}/cell/quicksettings/getSettingGroupTree.action",stringify(params)).then(function(res){
					vm.menuList = res.data;
					vm.menuList.map(function(item){
						item.show = true;
						item.disabled = disabled;
						
						if(item.text == 'LTE-TURBO') {
							item.show = isLWAEnable;
						}
						
						if(item.text == 'Quick Setting') {
							item.disabled = false;
						}
					});
					
					vm.activeCode = res.data[0].code;
					vm.loadPage(res.data[0].code, res.data[0].id, vm.sn, vm.connStatus)
				})
			},
			changeMenu(code,id,disabled){
				var vm = this;
				
				if(disabled !== true) {
					this.activeCode = code;
					this.loadPage(code,id, vm.sn, vm.connStatus);
				}
			},
			loadPage(code,id, sn, connStatus){
				var vm = this,
					method = 'get',
					urlList = {
						'Quick Setting' : '${ctx}/cell/quicksettings/goQuickSettingParamPage.action',
						'Network' : '${ctx}/cell/quicksettings/goNetWorkParamPage.action',
						'LTE' : '${ctx}/cell/quicksettings/goLTEParamPage.action',
						'BTS' : '${ctx}/cell/quicksettings/goBTSParamPage.action',
						'LTE-TURBO': '${ctx}/cell/ap/toAPInfo.action'
					},
					params = {
						smallCellCode: vm.smallCellCode,
                        enbSerialNumber: sn,
                        connectionStatus: connStatus
					};
				
				if(code == 'LTE-TURBO') {
					method = 'post';
				}
				
				$('#enbSettingRightPage').html('');
				loadHTML(document.querySelector('#enbSettingRightPage'),{
                    url: urlList[code],
                    method: method,
                    queryParams: params,
                    success: function() {
						setTimeout(function(){
        					eventBus.$emit('tab-param',vm.smallCellCode,id);
						}, 50);
                    }
                });
			},
			cancel(){
				eventBus.$emit('cancel-set-tab');
			}
		},
		mounted(){
			eventBus.$off('enb-set').$on('enb-set',this.init);
			eventBus.$off('cancel-set').$on('cancel-set',this.cancel);
		}
	})
</script>
<style>
	#enbSettingNewPanel .left-title-list li{
		height:36px;
		line-height:36px;
		padding:0 15px;
		cursor:pointer;
		border-bottom:1px solid #EEE;
		word-break:keep-all;
	}
	#enbSettingNewPanel .left-title-list li.active{
		color:#4D84FF;
		background:#EDF6FF;
		font-weight:bold;
	}
	
	#enbSettingNewPanel .el-collapse-item__header{
		border-bottom:1px solid #fff;
	}
	#enbSettingNewPanel .el-collapse-item__arrow{
		position:absolute;
		left:20px;
		top:-1px;
	}
	#enbSettingNewPanel .el-collapse-item{
		position:relative;
	}
	#enbSettingNewPanel .el-icon-arrow-right{
		font-size:16px;
	}
	#enbSettingNewPanel .el-icon-arrow-right:before{
		content:"\e639";
		color:#BBB;
	}
	#enbSettingNewPanel .is-active.el-icon-arrow-right:before{
		content:"\e638";
		color:#BBB;
	}
	#enbSettingNewPanel .el-collapse{
		border-top:1px solid #fff;
		border-bottom:1px solid #fff;
	}
	#enbSettingNewPanel .el-collapse-item__wrap{
		border-bottom:1px solid #EEE;
		padding-left:70px;
	}
	#enbSettingNewPanel .item-title-cls{
		font-weight:bold;
		position:relative;
		padding-left:15px;
		margin-bottom:20px;
	}
	#enbSettingNewPanel .item-title-cls:before{
		content:' ';
		width:6px;
		height:6px;
		background:#000;
		border-radius:6px;
		display:inline-block;
		position:absolute;
		top:8px;
		left:0px;
	}
	#enbSettingNewPanel .el-form-item{
		margin-bottom:20px;
		margin-left:16px;
		display:inline-block;
		width:45%;
	}
	#quickSettingPanel .el-form-item{
		margin-bottom:20px;
		margin-left:16px;
		display:inline-block;
		width:95%;
	}
	#enbSettingNewPanel .el-form-item__label{
		line-height:28px;
		font-size:12px;
		margin-left:15px;
	}
	#enbSettingNewPanel .list-cls{
		display:flex;
	}
	#enbSettingNewPanel .list-cls .el-form-item{
		width:100%;
	}
	#enbSettingNewPanel .list-item-cls{
		flex:1;
		display:flex;
		flex-direction:column;
	}
	#enbSettingNewPanel .suffixItem{
		display:inline-block;
		margin-bottom:0px;
		margin-right:5px;
	}
	#enbSettingNewPanel .suffixItem .el-form-item__content{
		line-height:16px;
	}
	#enbSettingNewPanel .suffixItem .form-suffix{
		width:85px;
	}
	#enbSettingNewPanel .suffixItem .form-suffix .text{
		width:55px;
	}
	#enbSettingNewPanel .el-form--inline .el-form-item{
		margin-right:0px;
		margin-left:0px !important;
	}
	#enbSettingNewPanel .item-tip{
		color:#999;
		margin-left:10px;
	}
	#enbSettingNewPanel .el-input-group__append{
		border-radius:0px;
		border-right:none;
		width:auto;
	}
	#enbSettingNewPanel .mmeSelect .el-input,#enbSettingNewPanel .mmeSelect .el-input__inner{
		width:100px;
	}
	.validate-item .el-input__inner{
		width:200px;
	}
	.validate-item .el-input-group__append{
		border:none;
		background:none;
	}
	.validate-item .el-form-item__error{
		display:none;
	}
	.is-error .el-input-group__append,.is-error .item-tip{
		color:#FA5555;
	}
	#enbSettingNewPanel .el-collapse-item__header{
		width:250px;
	}
	.addSlide{
		position:absolute;
		left:0px;
		right:0px;
		top:0px;
		bottom:0px;
		background:#fff;
		z-index:100;
	}
	#enbSettingNewPanel .el-form-item__error{
		width:200px;
		top:3px;
		left:210px;
	}
</style>